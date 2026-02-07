# DORA-yaki - Yet another DORA metrics dashboard

GitHub のメトリクスを収集し、開発生産性を可視化・分析するダッシュボードアプリケーションです。

> **English version**: [README.md](./README.md)

<img src="https://github.com/user-attachments/assets/3a7941d2-c18e-4742-b73f-8823351f7643" alt="DORA-yaki Top Page" width="800" >

## スクリーンショット

### ダッシュボード (トップページ)

DORAメトリクス、サイクルタイム内訳、レビュー分析、生産性スコアを一覧表示。

<img src="https://github.com/user-attachments/assets/3a7941d2-c18e-4742-b73f-8823351f7643" alt="DORA-yaki ダッシュボードページ" width="800" >

### メトリクス

日次トレンドチャート、PRサイズ分布、時系列分析などの詳細メトリクス。

<img src="https://github.com/user-attachments/assets/3a7941d2-c18e-4742-b73f-8823351f7643" alt="DORA-yaki メトリクスページ" width="600" >

### リポジトリ

GitHubリポジトリの登録と、1日〜1年の柔軟な期間でのデータ同期。

<img src="https://github.com/user-attachments/assets/711cd4e7-d0e8-45f6-a77c-e7e651b28d03" alt="DORA-yaki リポジトリページ" width="600" >

### チームパフォーマンス

メンバー別のPR数、レビュー活動、コード変更量を日次/週次ヒートマップチャートで表示。

<img src="https://github.com/user-attachments/assets/bd20d343-9e5b-491e-b421-920726fb3463" alt="DORA-yaki チームパフォーマンスページ" width="600" >


### メンバー詳細

個人の貢献者ビュー。PR履歴、レビュー履歴、アクティビティタイムラインを表示。

<img src="https://github.com/user-attachments/assets/911dfb62-a572-4676-80c0-c02bb2e48cc4" alt="DORA-yaki メンバー詳細ページ" width="600" >

### リポジトリ詳細

リポジトリ別のメトリクス内訳、PRリスト、サイクルタイムトレンド、コントリビューター統計。

<img src="https://github.com/user-attachments/assets/bec28942-1d88-431a-96dd-41f44fa9dcf0" alt="DORA-yaki リポジトリ詳細ページ" width="600" >

### ボット管理

ボットアカウントの識別・管理。ボット専用のPR・レビューメトリクスを人間の活動と分離して表示。

<img src="https://github.com/user-attachments/assets/839d6dc6-01ca-49b2-b822-85ec4764ad2f" alt="DORA-yaki ボット管理ページ" width="600" >

## 機能

- **DORA メトリクス** — デプロイ頻度、変更リードタイム、変更失敗率、平均復旧時間 (MTTR)
- **サイクルタイム分析** — コーディング / ピックアップ / レビュー / マージ の各工程と開発者別統計
- **レビュー分析** — レビュー数・コメント数、レビュアー別統計、初回レビュー時間
- **生産性スコア** — 複合スコア (0-100) と改善提案
- **チーム分析** — メンバー別の日次/週次チャートとPR・レビュー履歴
- **ボットユーザー管理** — ボットアカウントをメトリクスから除外、またはボット専用メトリクスを表示
- **多言語対応** — 8言語 (ja, en, zh-TW, zh-CN, ko, es, fr, de)、ブラウザ言語自動検出対応

## 技術スタック

| レイヤー | 技術 |
|---------|------|
| バックエンド | Go 1.25, net/http (stdlib) |
| フロントエンド | SvelteKit 2 (Svelte 5), Chart.js, date-fns |
| データベース | Google Cloud Datastore |
| インフラ | Google Cloud Functions / Cloud Run, Terraform |
| コンテナ | Distroless (debian12) |
| パッケージマネージャ | pnpm (フロントエンド) |
| 外部API | GitHub REST API |

## クイックスタート

### 前提条件

- Go 1.25+
- Node.js 24+
- pnpm
- Docker & Docker Compose
- GitHub Personal Access Token
- Google Cloud プロジェクト (本番環境)

### ローカル開発

1. リポジトリをクローン:
```bash
git clone https://github.com/compasstechlab/dora-yaki.git
cd dora-yaki
```

2. 環境変数を設定:
```bash
cp backend/.env.example backend/.env
# backend/.env を編集して GitHub トークンと GCP プロジェクト ID を設定

cp frontend/.env.example frontend/.env
# frontend/.env を編集してデフォルト言語や API URL を変更（必要に応じて）
```

3. Docker Compose で起動:
```bash
docker compose up
```

4. アプリケーションにアクセス:
- アプリケーション: http://localhost:7201

### 手動セットアップ

**バックエンド:**
```bash
cd backend
go mod download
GITHUB_TOKEN=your_token go run ./cmd/httpserver/main.go
```

**フロントエンド:**
```bash
cd frontend
pnpm install
pnpm run dev
```

## API エンドポイント

### ヘルスチェック
- `GET /health` - ヘルスチェック

### キャッシュ
- `POST /api/cache/invalidate` - レスポンスキャッシュの全クリア

### リポジトリ
- `GET /api/repositories` - リポジトリ一覧
- `POST /api/repositories` - リポジトリ追加
- `GET /api/repositories/{id}` - リポジトリ取得
- `DELETE /api/repositories/{id}` - リポジトリ削除
- `POST /api/repositories/batch` - リポジトリ一括追加
- `POST /api/repositories/{id}/sync` - リポジトリデータ同期
- `GET /api/repositories/date-ranges` - リポジトリの日付範囲取得

### GitHub
- `GET /api/github/me` - 認証済み GitHub ユーザー取得
- `GET /api/github/owners/{owner}/repos` - オーナーのリポジトリ一覧

### メトリクス
- `GET /api/metrics/cycle-time` - サイクルタイム分析
- `GET /api/metrics/reviews` - レビュー分析
- `GET /api/metrics/dora` - DORA メトリクス
- `GET /api/metrics/productivity-score` - 生産性スコア
- `GET /api/metrics/daily` - 日次集約メトリクス
- `GET /api/metrics/pull-requests` - プルリクエスト一覧

### スプリント
- `GET /api/sprints` - スプリント一覧
- `POST /api/sprints` - スプリント作成
- `GET /api/sprints/{id}` - スプリント取得
- `GET /api/sprints/{id}/performance` - スプリントパフォーマンス

### ボットユーザー
- `GET /api/bot-users` - ボットユーザー一覧
- `POST /api/bot-users` - ボットユーザー追加
- `DELETE /api/bot-users` - ボットユーザー削除

### チーム
- `GET /api/team/members` - チームメンバー一覧
- `GET /api/team/members/{id}/stats` - メンバー統計
- `GET /api/team/members/{id}/pull-requests` - メンバーのプルリクエスト
- `GET /api/team/members/{id}/reviews` - メンバーのレビュー

### ジョブ
- `PUT /api/job/sync` - データ同期ジョブの実行

## プロジェクト構成

```
dora-yaki/
├── backend/
│   ├── cmd/httpserver/           # エントリポイント
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handler/         # HTTP ハンドラー (metrics, repository, team, job 等)
│   │   │   ├── middleware/       # CORS, ロガー, キャッシュ
│   │   │   └── router.go        # ルート定義
│   │   ├── config/              # 設定
│   │   ├── datastore/           # Cloud Datastore クライアント
│   │   ├── domain/model/        # ドメインモデル
│   │   ├── github/              # GitHub API クライアント & コレクター
│   │   ├── metrics/             # 計算 & 集約
│   │   └── timeutil/            # タイムゾーンオフセット処理
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── api/             # API クライアント
│   │   │   ├── components/      # Svelte コンポーネント (PeriodSelector 等)
│   │   │   ├── i18n/            # 多言語対応
│   │   │   ├── stores/          # Svelte ストア
│   │   │   └── utils/           # ユーティリティ関数
│   │   └── routes/              # SvelteKit ページ
│   ├── Dockerfile
│   └── package.json
├── terraform/                   # Infrastructure as Code
├── docs/
│   └── ARCHITECTURE.md
├── compose.yml
└── README.md
```

## 環境変数

| 変数名 | 説明 | 必須 |
|--------|------|------|
| `GITHUB_TOKEN` | GitHub Personal Access Token | はい |
| `GCP_PROJECT_ID` | Google Cloud プロジェクト ID | はい (開発用) |
| `PORT` | バックエンドのポート (デフォルト: 7202) | いいえ |
| `ENVIRONMENT` | development / production | いいえ |
| `TZ_OFFSET` | タイムゾーンオフセット (例: `+09:00`, `-05:30`)。未設定時は UTC | いいえ |
| `FUNCTION_TARGET` | Cloud Functions エントリポイント (デフォルト: `RunHTTPServer`) | いいえ |
| `API_BACKEND` | サーバーサイドプロキシのバックエンド API URL (デフォルト: `http://localhost:7202`) | いいえ |
| `VITE_API_BASE` | バックエンド API のベースパス (フロントエンド、デフォルト: `/api`) | いいえ |
| `VITE_DEFAULT_LOCALE` | デフォルト言語 (デフォルト: `ja`) | いいえ |

### GitHub トークンの設定

DORA-yaki はリポジトリデータ、プルリクエスト、レビュー、デプロイメントを取得するために GitHub Personal Access Token が必要です。

#### 方法 1: Fine-grained Personal Access Token (推奨)

Fine-grained PAT はより細かい権限制御が可能で、GitHub が推奨する方式です。

1. [GitHub Settings > Developer settings > Fine-grained tokens](https://github.com/settings/personal-access-tokens/new) にアクセス
2. **Token name** と **Expiration** を設定
3. **Resource owner** でアクセスしたいOrganizationを選択 (例: `your-org`)
   - Organizationが一覧に表示されない場合、org 管理者が [Organization settings > Personal access tokens](https://github.com/organizations/YOUR_ORG/settings/personal-access-tokens) で Fine-grained PAT のアクセスを許可する必要があります
4. **Repository access** で **All repositories** を選択、または特定のリポジトリを選択
5. **Permissions > Repository permissions** で以下の権限を付与:

| 権限 | アクセスレベル | 用途 |
|------|-------------|------|
| **Metadata** | Read | リポジトリ一覧の取得 |
| **Contents** | Read | リポジトリデータの読み取り |
| **Pull requests** | Read | サイクルタイム・レビューメトリクス用のPRデータ取得 |
| **Deployments** | Read | DORAデプロイ頻度用のデプロイメントデータ取得 |

6. **Generate token** をクリックしてトークンを `backend/.env` にコピー

#### 方法 2: Classic Personal Access Token

1. [GitHub Settings > Developer settings > Tokens (classic)](https://github.com/settings/tokens/new) にアクセス
2. 以下のスコープを選択:

| スコープ | 用途 |
|---------|------|
| `repo` | リポジトリへのフルアクセス (PR、デプロイメント含む) |
| `read:org` | Organizationのメンバーシップ・リポジトリ一覧の読み取り |

3. **Generate token** をクリックしてトークンを `backend/.env` にコピー

> **注意**: Organizationのリポジトリにアクセスする場合、トークンの所有者がそのOrganizationのメンバーである必要があります。Fine-grained PAT を使用する場合、Organization 管理者が Fine-grained PAT のアクセスを許可している必要があります。

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

1. [Terraform](https://developer.hashicorp.com/terraform/install) をインストール (>= 1.0)

2. 変数を設定:
```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
# terraform.tfvars を編集して GCP プロジェクト ID とリージョンを設定
```

| 変数名 | 説明 | デフォルト |
|--------|------|-----------|
| `project_id` | GCP プロジェクト ID | (必須) |
| `region` | GCP リージョン | `asia-northeast1` |

3. 初期化と適用:
```bash
terraform init
terraform plan    # 変更内容を確認
terraform apply   # 変更を適用
```

4. 作成されたシークレットに GitHub トークンの値を追加:
```bash
echo -n "your_github_token" | gcloud secrets versions add GITHUB_TOKEN --data-file=-
```

> **注意**: Terraform はシークレットの定義（空データ）のみを作成します。実際のトークン値は `gcloud` コマンドまたは GCP コンソールから設定してください。

## セキュリティに関する注意事項

> **重要**: このアプリケーションには認証・認可機能が**組み込まれていません**。全てのAPIエンドポイントはデフォルトで公開されています。本番環境にデプロイする際は、外部の認証レイヤーでフロントエンドとバックエンドの両方を保護する**必要があります**。

推奨される方法:

- [Cloud IAP (Identity-Aware Proxy)](https://cloud.google.com/iap) - GCP デプロイ向け
- [Cloudflare Access](https://www.cloudflare.com/products/zero-trust/access/) - Cloudflare ベースのデプロイ向け
- [Google Cloud Armor (WAF)](https://cloud.google.com/armor) - GCP 向け Web Application Firewall
- OAuth2 Proxy / 認証付きリバースプロキシ

> **注意**: デフォルトの CORS 設定はローカル開発の利便性のため全オリジン (`*`) を許可しています。本番環境では `backend/internal/api/router.go` を修正し、フロントエンドのドメインのみを許可してください。

## デプロイ

### バックエンド (Cloud Functions gen2)

バックエンドをデプロイ:
```bash
cd backend
make deploy
```

### フロントエンド

フロントエンドは以下のいずれかにデプロイ可能:

- **Cloud Run**: `cd frontend && make deploy-cloudrun`
- **Cloudflare Pages**: `cd frontend && make deploy-cloudflare`
- **Firebase Hosting**: `cd frontend && make deploy-firebase`

### Cloud Scheduler (任意)

GitHub データを定期的に自動同期するジョブを設定:

```bash
cd backend
make schedule-create project=your-gcp-project region=asia-northeast1
```

スケジュールを更新する場合:
```bash
make schedule-update project=your-gcp-project region=asia-northeast1
```

## Vibe Coding

このプロジェクトは **Vibe Coding** — [Claude Code](https://claude.com/claude-code) を活用した AI 支援開発 — によってほぼ全て構築されました。

- アーキテクチャ設計、バックエンド/フロントエンド実装、Terraform IaC、ドキュメントの全てを AI ペアプログラミングで生成
- 人間の役割: 要件定義、設計判断、コードレビュー、最終承認
- AI の役割: コード生成、リファクタリング、テスト、ドキュメント作成

## ライセンス

このプロジェクトは [GNU Affero General Public License v3.0 (AGPL-3.0)](./LICENSE) の下でライセンスされています。

**AGPL-3.0 第13条**に基づき、このソフトウェアを修正してネットワーク経由で提供する場合、リモートでアクセスする全てのユーザーに対応するソースコードへのアクセスを提供する必要があります。アプリケーションの UI フッターにソースコードへのリンクを表示することで、この要件に対応しています。
