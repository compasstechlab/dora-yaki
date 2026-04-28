# デプロイガイド

## 前提条件

- [Terraform](https://developer.hashicorp.com/terraform/install) (>= 1.0)
- Google Cloud プロジェクト
- GitHub OAuth App ([GitHub OAuth App の設定](#github-oauth-app-の設定) を参照)

## GitHub OAuth App の設定

dora-yaki は全ユーザーを GitHub OAuth で認証します。共有の GitHub
Personal Access Token は不要になりました。

1. <https://github.com/settings/developers> -> **OAuth Apps** -> **New OAuth App** を開く
2. 以下を入力:
    - **Application name**: 例 `dora-yaki (production)`
    - **Homepage URL**: フロントエンドの origin、例 `https://dora-yaki.example.com`
    - **Authorization callback URL**: `https://<backend-domain>/api/auth/github/callback`
3. **Register application** をクリック
4. **Client ID** をコピーし、**Generate a new client secret** で Client Secret を発行する
5. (推奨) OAuth App 設定の **Expire user authorization tokens** を有効にして refresh token が発行されるようにする

OAuth フローでは `read:user`, `user:email`, `repo` のスコープを要求します。
`repo` はログインユーザーが GitHub 上で閲覧権限を持つ private リポジトリの
メタデータを取得するために必要です。

> 各ユーザーは自分の GitHub アカウントでログインします。dora-yaki は
> 取得した OAuth トークンを Datastore に **暗号化** して保存し
> (AES-256-GCM または Cloud KMS)、private リポジトリの sync 時に
> あるユーザーのトークンが失敗した場合は別ユーザーのトークンに自動でローテーションします。

## インフラ (Terraform)

GCP リソースは Terraform (`terraform/` ディレクトリ) で管理しています。

### 管理リソース

| リソース | 説明 |
|----------|------|
| GCP API | Cloud Functions, Cloud Run, Cloud Build, Cloud Scheduler, Secret Manager, Cloud KMS, Firestore |
| サービスアカウント | `dora-yaki-api` (Datastore アクセス, Secret Manager アクセス, KMS encrypter/decrypter (任意), Cloud Run invoker) |
| Secret Manager | `GITHUB_OAUTH_CLIENT_ID`, `GITHUB_OAUTH_CLIENT_SECRET`, `AUTH_JWT_SECRET`, `ENCRYPTION_KEY_BASE64`, `JOB_AUTH_KEY` |
| Cloud KMS (任意) | キーリング `dora-yaki`、キー `tokens` (90日ローテーション)。 `use_kms = true` の場合のみ作成 |
| Cloud Scheduler | Terraform 管理の sync ジョブと permission-check ジョブ |
| Datastore インデックス | PullRequest, Review, Deployment, DailyMetrics, Sprint, RepositoryAccess のコンポジットインデックス |

### セットアップ

1. 変数を設定:
```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
# terraform.tfvars に GCP プロジェクト ID、リージョン、リソース名、
# 必要に応じて backend_service_url や use_kms を設定
```

| 変数名 | 説明 | デフォルト |
|--------|------|-----------|
| `project_id` | GCP プロジェクト ID | (必須) |
| `region` | GCP リージョン | `asia-northeast1` |
| `resource_name_prefix` | Scheduler ジョブなどの生成リソース名の prefix | `dora-yaki` |
| `service_account_id` | バックエンド用サービスアカウント ID | `dora-yaki-api` |
| `service_account_display_name` | バックエンド用サービスアカウントの表示名 | `DORA-Yaki API` |
| `secret_replication_locations` | Secret Manager の user-managed replication ロケーション。空なら自動 replication | `[]` |
| `create_datastore_database` | Datastore mode の default Firestore database を作成するか | `true` |
| `datastore_location_id` | Datastore mode database のロケーション。空なら `region` | `""` |
| `backend_service_url` | Cloud Scheduler が叩くバックエンドの絶対URL。初回デプロイ後に設定 | `""` (scheduler スキップ) |
| `scheduler_auth_key` | Cloud Scheduler が `X-DORA-YAKI-JOB-KEY` で送る共有キー。`JOB_AUTH_KEY` と同じ値 | `""` (scheduler スキップ) |
| `use_kms` | トークン暗号化用の Cloud KMS キーを作成するか | `false` |
| `kms_location` | KMS キーリングのロケーション | `asia-northeast1` |
| `kms_key_ring_name` | KMS キーリング名 | `dora-yaki` |
| `kms_crypto_key_name` | KMS crypto key 名 | `tokens` |
| `sync_job_name` | sync 用 Cloud Scheduler ジョブ名。空なら `<resource_name_prefix>-sync` | `""` |
| `sync_schedule` | リポジトリ sync ジョブの cron | `*/15 * * * *` |
| `permission_check_job_name` | 権限チェック用 Cloud Scheduler ジョブ名。空なら `<resource_name_prefix>-permission-check` | `""` |
| `permission_check_schedule` | 日次 ACL 更新ジョブの cron | `0 3 * * *` |
| `scheduler_time_zone` | Cloud Scheduler の IANA タイムゾーン | `Asia/Tokyo` |

2. 初期化と適用:
```bash
terraform init
terraform plan    # 変更内容を確認
terraform apply   # 変更を適用
```

3. ランタイムシークレットを生成して投入:

```bash
# ランダムシークレット
JWT_SECRET=$(openssl rand -base64 32)
ENC_KEY=$(openssl rand -base64 32)
JOB_KEY=$(openssl rand -base64 32)

# Secret Manager に投入 (Terraform はシークレットの "枠" のみ作成、値は別途投入)
gcloud secrets versions add GITHUB_OAUTH_CLIENT_ID    --data-file=- <<<"<OAuth App から>"
gcloud secrets versions add GITHUB_OAUTH_CLIENT_SECRET --data-file=- <<<"<OAuth App から>"
gcloud secrets versions add AUTH_JWT_SECRET           --data-file=- <<<"$JWT_SECRET"
gcloud secrets versions add ENCRYPTION_KEY_BASE64     --data-file=- <<<"$ENC_KEY"
gcloud secrets versions add JOB_AUTH_KEY              --data-file=- <<<"$JOB_KEY"
```

> **注意**: Terraform はシークレットの定義（空データ）のみを作成します。実際の値は `gcloud` または GCP コンソールから設定してください。5 つすべてを機微情報として扱ってください。
> Secret ID は backend の環境変数名と同じ固定値にしています。`backend/Makefile`
> の `--set-secrets` も同じ名前で固定しています。

> **Cloud KMS を使う場合**: `use_kms = true` を設定した場合は
> `ENCRYPTION_KEY_BASE64` の代わりに、バックエンドの `ENCRYPTION_KMS_KEY`
> 環境変数に作成された鍵のリソースパス
> (`terraform output -raw kms_crypto_key_id`、または
> `projects/<project>/locations/<location>/keyRings/<kms_key_ring_name>/cryptoKeys/<kms_crypto_key_name>`)
> を設定してください。`backend/Makefile` は deploy 用変数 `BACKEND_ENCRYPTION_KMS_KEY` が
> 設定されている場合はランタイムの `ENCRYPTION_KMS_KEY` として渡し、未設定の場合は
> `ENCRYPTION_KEY_BASE64` の Secret Manager secret を紐づけます。
> ランタイムで両方設定された場合は KMS が優先されます。

## バックエンド (Cloud Functions gen2)

ランタイムに以下の環境変数を設定:

| 変数名 | 取得元 | 備考 |
|--------|--------|------|
| `GITHUB_OAUTH_CLIENT_ID` | Secret Manager | |
| `GITHUB_OAUTH_CLIENT_SECRET` | Secret Manager | |
| `AUTH_JWT_SECRET` | Secret Manager | |
| `ENCRYPTION_KEY_BASE64` | Secret Manager | KMS 利用時は省略 |
| `JOB_AUTH_KEY` | Secret Manager | 本番必須。`/api/job/*` 用の共有キー |
| `ENCRYPTION_KMS_KEY` | env (平文) | KMS リソースパス。base64 利用時は省略 |
| `OAUTH_REDIRECT_URL` | env (平文) | `https://<backend-domain>/api/auth/github/callback` |
| `FRONTEND_URL` | env (平文) | フロントエンドの origin (ログイン後のリダイレクト先) |
| `AUTH_COOKIE_DOMAIN` | env (平文) | 任意。フロントエンドとバックエンドが同一ドメインのサブドメイン構成のときに設定 |
| `GCP_PROJECT_ID` | env (平文) | |
| `TZ_OFFSET` | env (平文) | 任意 |

デプロイ:

```bash
cd backend
make deploy
```

KMS 利用時は `make deploy` 前に `BACKEND_ENCRYPTION_KMS_KEY` を設定してください:

```bash
export BACKEND_ENCRYPTION_KMS_KEY="$(terraform -chdir=../terraform output -raw kms_crypto_key_id)"
make deploy
```

`backend/Makefile` は `../.env` がある場合だけ読み込みます。無い場合は現在の shell 環境変数を使います。
Terraform 側の名前を変更した場合は `.env` または export 済みの環境変数に同じ値を設定してください。
例えば Terraform の `service_account_id` は `BACKEND_SERVICE_ACCOUNT_ID` と一致させてください。
Secret Manager ID は `--set-secrets` で使う backend 環境変数名と同じ固定名です。
デプロイ前に解決済みの変数を表示し、`Continue? [y/N]` で確認します。

初回デプロイ後にサービス URL を取得し、`terraform.tfvars` の
`backend_service_url` と `scheduler_auth_key` に設定して再度
`terraform apply` すると Cloud Scheduler ジョブが作成されます。
`scheduler_auth_key` は Secret Manager の `JOB_AUTH_KEY` と同じ値にしてください。
この値は Scheduler の HTTP ヘッダーとして Terraform state に保存されるため、
state backend を適切に保護してください。

## フロントエンド

フロントエンドは以下のいずれかにデプロイ可能:

- **Cloud Run**: `cd frontend && make deploy-cloudrun`
- **Cloudflare Pages**: `cd frontend && make deploy-cloudflare`
- **Firebase Hosting**: `cd frontend && make deploy-firebase`

`frontend/Makefile` も `../.env` がある場合だけ読み込み、無い場合は現在の shell 環境変数を使います。
必要なデプロイ変数を validate してから実行します。
フロントエンドは `/api/*` をバックエンドに転送します。必要に応じて `API_BACKEND` に
バックエンドの絶対 URL を設定してください。
frontend の deploy ターゲットもデプロイ前に解決済みの変数を表示し、`Continue? [y/N]` で確認します。

## Cloud Scheduler

`backend_service_url` と `scheduler_auth_key` を設定すると Terraform で 2 つのジョブが作成されます:

- `<resource_name_prefix>-sync` (デフォルト `*/15 * * * *`) - `PUT /api/job/sync` を呼ぶ
- `<resource_name_prefix>-permission-check` (デフォルト `0 3 * * *`) - `PUT /api/job/permission-check` を呼ぶ

両ジョブは `X-DORA-YAKI-JOB-KEY` と、設定したバックエンド用サービスアカウントで発行した
OIDC トークンを送ります。アプリケーションは共有ジョブキーを検証し、OIDC トークンは
Cloud Run / Cloud Functions の IAM 呼び出し互換性のために使います。

## セキュリティに関する注意事項

GitHub OAuth によるユーザー認証が組み込まれました。`/api/auth/*` と
`/api/cache/invalidate` を除くすべての `/api/*` エンドポイントは OAuth
コールバックで発行されるセッションクッキーが必要です。

本番環境で追加で検討すべき事項:

- `FRONTEND_URL` を実際のフロントエンド origin に設定し、CORS の許可 origin を限定する
- `/api/job/*` を保護する `JOB_AUTH_KEY` を設定し、定期的にローテーションする
- `AUTH_COOKIE_DOMAIN` を設定してセッションクッキーのスコープを限定する
- `AUTH_JWT_SECRET` / `ENCRYPTION_KEY_BASE64` の定期ローテーション (または Cloud KMS の 90日自動ローテーションを利用)
- GitHub OAuth App 設定の **Restrict to organization** で利用者を組織内に限定
- バックエンドの前段に [Cloud Armor](https://cloud.google.com/armor) を配置して OAuth エンドポイントへの攻撃を緩和

## 旧 GITHUB_TOKEN 構成からの移行

旧バージョンでは共有の `GITHUB_TOKEN` 環境変数を使用していました。移行手順:

1. 新しい Terraform を `apply` する (5 つのシークレット枠と必要に応じて KMS リソースが作成される)
2. `AUTH_JWT_SECRET` / `ENCRYPTION_KEY_BASE64` / `JOB_AUTH_KEY` を生成し、OAuth App の Client ID / Secret と一緒に Secret Manager に投入する
3. 新しい環境変数でバックエンドを再デプロイする (`GITHUB_TOKEN` はもう参照されない)。`backend/Makefile` は `--set-secrets` を使うため、古い secret 環境変数は現在の一覧で置き換わる
4. 最初にログインしたユーザーが `RepositoryAccess` のシードになります。既存のリポジトリデータはそのまま残り、各ユーザーの初回ログインと日次 permission-check ジョブで権限が反映されます
5. 旧 `GITHUB_TOKEN` シークレットは参照がなくなったら削除する
