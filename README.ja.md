# DORA-yaki - Yet another DORA metrics dashboard

GitHub のメトリクスを収集し、開発生産性を可視化・分析するダッシュボードアプリケーションです。

> **English version**: [README.md](./README.md)

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

- Docker & Docker Compose
- GitHub Personal Access Token ([設定ガイド](./docs/DEPLOYMENT.ja.md#github-トークンの設定))

### Docker Compose で起動

```bash
git clone https://github.com/compasstechlab/dora-yaki.git
cd dora-yaki

cp .env.example .env
# .env を編集して GitHub トークンと GCP プロジェクト ID を設定

docker compose up
```

アプリケーション: http://localhost:7201

## ドキュメント

| ドキュメント | 説明 |
|-------------|------|
| [デプロイガイド](./docs/DEPLOYMENT.ja.md) | 本番デプロイ (Terraform, Cloud Functions, Cloud Run, セキュリティ) |
| [開発ガイド](./docs/DEVELOPMENT.md) | Docker なしのローカルセットアップ、アーキテクチャ、API エンドポイント、プロジェクト構成 |

## Vibe Coding

このプロジェクトは **Vibe Coding** — [Claude Code](https://claude.com/claude-code) を活用した AI 支援開発 — によってほぼ全て構築されました。

- アーキテクチャ設計、バックエンド/フロントエンド実装、Terraform IaC、ドキュメントの全てを AI ペアプログラミングで生成
- 人間の役割: 要件定義、設計判断、コードレビュー、最終承認
- AI の役割: コード生成、リファクタリング、テスト、ドキュメント作成

## ライセンス

このプロジェクトは [GNU Affero General Public License v3.0 (AGPL-3.0)](./LICENSE) の下でライセンスされています。
