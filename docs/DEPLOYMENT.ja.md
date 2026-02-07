# デプロイガイド

## 前提条件

- [Terraform](https://developer.hashicorp.com/terraform/install) (>= 1.0)
- Google Cloud プロジェクト
- GitHub Personal Access Token ([GitHub トークンの設定](#github-トークンの設定) を参照)

## インフラ (Terraform)

GCP リソースは Terraform (`terraform/` ディレクトリ) で管理しています。

### 管理リソース

| リソース | 説明 |
|----------|------|
| GCP API | Cloud Functions, Cloud Run, Cloud Build, Cloud Scheduler (GitHub データの定期同期), Secret Manager |
| サービスアカウント | `dora-yaki-api` (Datastore アクセス, Secret Manager アクセス) |
| Secret Manager | `GITHUB_TOKEN` シークレットシェル |
| Datastore インデックス | PullRequest, Review, Deployment, DailyMetrics, Sprint のコンポジットインデックス |

### セットアップ

1. 変数を設定:
```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
# terraform.tfvars を編集して GCP プロジェクト ID とリージョンを設定
```

| 変数名 | 説明 | デフォルト |
|--------|------|-----------|
| `project_id` | GCP プロジェクト ID | (必須) |
| `region` | GCP リージョン | `asia-northeast1` |

2. 初期化と適用:
```bash
terraform init
terraform plan    # 変更内容を確認
terraform apply   # 変更を適用
```

3. 作成されたシークレットに GitHub トークンの値を追加:
```bash
echo -n "your_github_token" | gcloud secrets versions add GITHUB_TOKEN --data-file=-
```

> **注意**: Terraform はシークレットの定義（空データ）のみを作成します。実際のトークン値は `gcloud` コマンドまたは GCP コンソールから設定してください。

## バックエンド (Cloud Functions gen2)

```bash
cd backend
make deploy
```

## フロントエンド

フロントエンドは以下のいずれかにデプロイ可能:

- **Cloud Run**: `cd frontend && make deploy-cloudrun`
- **Cloudflare Pages**: `cd frontend && make deploy-cloudflare`
- **Firebase Hosting**: `cd frontend && make deploy-firebase`

## Cloud Scheduler (任意)

GitHub データを定期的に自動同期するジョブを設定:

```bash
cd backend
make schedule-create project=your-gcp-project region=asia-northeast1
```

スケジュールを更新する場合:
```bash
make schedule-update project=your-gcp-project region=asia-northeast1
```

## セキュリティに関する注意事項

> **重要**: このアプリケーションには認証・認可機能が**組み込まれていません**。全ての API エンドポイントはデフォルトで公開されています。本番環境にデプロイする際は、外部の認証レイヤーでフロントエンドとバックエンドの両方を保護する**必要があります**。

推奨される方法:

- [Cloud IAP (Identity-Aware Proxy)](https://cloud.google.com/iap) - GCP デプロイ向け
- [Cloudflare Access](https://www.cloudflare.com/products/zero-trust/access/) - Cloudflare ベースのデプロイ向け
- [Google Cloud Armor (WAF)](https://cloud.google.com/armor) - GCP 向け Web Application Firewall
- OAuth2 Proxy / 認証付きリバースプロキシ

> **注意**: デフォルトの CORS 設定はローカル開発の利便性のため全オリジン (`*`) を許可しています。本番環境では `backend/internal/api/router.go` を修正し、フロントエンドのドメインのみを許可してください。

## GitHub トークンの設定

DORA-yaki はリポジトリデータ、プルリクエスト、レビュー、デプロイメントを取得するために GitHub Personal Access Token が必要です。

### 方法 1: Fine-grained Personal Access Token (推奨)

Fine-grained PAT はより細かい権限制御が可能で、GitHub が推奨する方式です。

1. [GitHub Settings > Developer settings > Fine-grained tokens](https://github.com/settings/personal-access-tokens/new) にアクセス
2. **Token name** と **Expiration** を設定
3. **Resource owner** でアクセスしたい Organization を選択 (例: `your-org`)
   - Organization が一覧に表示されない場合、org 管理者が [Organization settings > Personal access tokens](https://github.com/organizations/YOUR_ORG/settings/personal-access-tokens) で Fine-grained PAT のアクセスを許可する必要があります
4. **Repository access** で **All repositories** を選択、または特定のリポジトリを選択
5. **Permissions > Repository permissions** で以下の権限を付与:

| 権限 | アクセスレベル | 用途 |
|------|-------------|------|
| **Metadata** | Read | リポジトリ一覧の取得 |
| **Pull requests** | Read | サイクルタイム・レビューメトリクス用の PR データ取得 |
| **Deployments** | Read | DORA デプロイ頻度用のデプロイメントデータ取得 |

6. **Generate token** をクリックしてトークンを `.env` にコピー

### 方法 2: Classic Personal Access Token

1. [GitHub Settings > Developer settings > Tokens (classic)](https://github.com/settings/tokens/new) にアクセス
2. 以下のスコープを選択:

| スコープ | 用途 |
|---------|------|
| `repo` | リポジトリへのフルアクセス (PR、デプロイメント含む) |
| `read:org` | Organization のメンバーシップ・リポジトリ一覧の読み取り |

3. **Generate token** をクリックしてトークンを `.env` にコピー

> **注意**: Organization のリポジトリにアクセスする場合、トークンの所有者がその Organization のメンバーである必要があります。Fine-grained PAT を使用する場合、Organization 管理者が Fine-grained PAT のアクセスを許可している必要があります。
