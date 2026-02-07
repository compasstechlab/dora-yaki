# DORA-yaki - Yet another DORA metrics dashboard

A dashboard application that collects GitHub metrics and visualizes development productivity.

> **日本語版**: [README.ja.md](./README.ja.md)

## Screenshots

### Dashboard (Top Page)

DORA metrics, cycle time breakdown, review analysis, and productivity score at a glance.

<img src="https://github.com/user-attachments/assets/3a7941d2-c18e-4742-b73f-8823351f7643" alt="DORA-yaki Dashboard Page" width="800" >

### Metrics

Detailed metrics with daily trend charts, PR size distribution, and time-series analysis.

<img src="https://github.com/user-attachments/assets/3a7941d2-c18e-4742-b73f-8823351f7643" alt="DORA-yaki Metrics Page" width="600" >

### Repositories

Register GitHub repositories and sync data with flexible time ranges (1 day to 1 year).

<img src="https://github.com/user-attachments/assets/711cd4e7-d0e8-45f6-a77c-e7e651b28d03" alt="DORA-yaki Repositories Page" width="600" >

### Team Performance

Per-member statistics including PR count, review activity, and code changes with daily/weekly heatmap charts.

<img src="https://github.com/user-attachments/assets/bd20d343-9e5b-491e-b421-920726fb3463" alt="DORA-yaki Team Performance Page" width="600" >


### Member Detail

Individual contributor view with PR history, review history, and activity timeline.

<img src="https://github.com/user-attachments/assets/911dfb62-a572-4676-80c0-c02bb2e48cc4" alt="DORA-yaki Member Detail Page" width="600" >

### Repository Detail

Per-repository metrics breakdown with PR list, cycle time trends, and contributor statistics.

<img src="https://github.com/user-attachments/assets/bec28942-1d88-431a-96dd-41f44fa9dcf0" alt="DORA-yaki Repository Detail Page" width="600" >

### Bot Management

Identify and manage bot accounts. View bot-specific PR and review metrics separately from human activity.

<img src="https://github.com/user-attachments/assets/839d6dc6-01ca-49b2-b822-85ec4764ad2f" alt="DORA-yaki Bot Management Page" width="600" >

## Features

- **DORA Metrics** — Deployment Frequency, Lead Time for Changes, Change Failure Rate, MTTR
- **Cycle Time Analysis** — Coding / Pickup / Review / Merge time breakdown with per-author statistics
- **Review Analysis** — Review and comment counts, per-reviewer statistics, first review time
- **Productivity Score** — Composite score (0-100) with improvement recommendations
- **Team Analytics** — Per-member statistics with daily/weekly charts and PR/review history
- **Bot User Management** — Exclude bot accounts from metrics or view bot-only metrics
- **Multi-language Support** — 8 languages (ja, en, zh-TW, zh-CN, ko, es, fr, de) with browser auto-detection

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.25, net/http (stdlib) |
| Frontend | SvelteKit 2 (Svelte 5), Chart.js, date-fns |
| Database | Google Cloud Datastore |
| Infrastructure | Google Cloud Functions / Cloud Run, Terraform |
| Container | Distroless (debian12) |
| Package Manager | pnpm (frontend) |
| External API | GitHub REST API |

## Quick Start

### Prerequisites

- Docker & Docker Compose
- GitHub Personal Access Token ([setup guide](./docs/DEPLOYMENT.md#github-token-setup))

### Run with Docker Compose

```bash
git clone https://github.com/compasstechlab/dora-yaki.git
cd dora-yaki

cp .env.example .env
# Edit .env with your GitHub token and GCP project ID

docker compose up
```

Access the application at http://localhost:7201

## Documentation

| Document | Description |
|----------|-------------|
| [Deployment Guide](./docs/DEPLOYMENT.md) | Production deployment (Terraform, Cloud Functions, Cloud Run, security) |
| [Development Guide](./docs/DEVELOPMENT.md) | Local setup without Docker, architecture, API endpoints, project structure |

## Vibe Coding

This project was built almost entirely through **Vibe Coding** — AI-assisted development using [Claude Code](https://claude.com/claude-code).

- Architecture design, backend/frontend implementation, Terraform IaC, and documentation were all generated through AI pair programming
- Human role: requirements definition, design decisions, code review, and final approval
- AI role: code generation, refactoring, testing, and documentation

## License

This project is licensed under the [GNU Affero General Public License v3.0 (AGPL-3.0)](./LICENSE).
