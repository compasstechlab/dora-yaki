# DORA-yaki - Yet another DORA metrics dashboard

A dashboard application that collects GitHub metrics and visualizes development productivity.

> **日本語版**: [README.ja.md](./README.ja.md)

## Features

- **DORA Metrics**
  - Deployment Frequency
  - Lead Time for Changes
  - Change Failure Rate
  - Mean Time to Recovery (MTTR)

- **Cycle Time Analysis**
  - Coding / Pickup / Review / Merge time breakdown
  - Per-author statistics

- **Review Analysis**
  - Review and comment counts
  - Per-reviewer statistics
  - First review time

- **Productivity Score**
  - Composite score (0-100)
  - Improvement recommendations

- **Team Analytics**
  - Per-member statistics with daily/weekly charts
  - PR and review history

- **Bot User Management**
  - Exclude bot accounts from metrics

- **Multi-language Support**
  - 8 languages: ja, en, zh-TW, zh-CN, ko, es, fr, de
  - Browser language auto-detection with localStorage persistence

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

- Go 1.25+
- Node.js 24+
- pnpm
- Docker & Docker Compose
- GitHub Personal Access Token
- Google Cloud Project (for production)

### Local Development

1. Clone the repository:
```bash
git clone https://github.com/compasstechlab/dora-yaki.git
cd dora-yaki
```

2. Set up environment variables:
```bash
cp backend/.env.example backend/.env
# Edit backend/.env with your GitHub token and GCP project ID

cp frontend/.env.example frontend/.env
# Edit frontend/.env if you need to change the default locale or API URL
```

3. Start with Docker Compose:
```bash
docker compose up
```

4. Access the application:
- Frontend: http://localhost:7201
- Backend API: http://localhost:7202

### Manual Setup

**Backend:**
```bash
cd backend
go mod download
GITHUB_TOKEN=your_token go run ./cmd/httpserver/main.go
```

**Frontend:**
```bash
cd frontend
pnpm install
pnpm run dev
```

## API Endpoints

### Health
- `GET /health` - Health check

### Cache
- `POST /api/cache/invalidate` - Clear all response cache

### Repositories
- `GET /api/repositories` - List repositories
- `POST /api/repositories` - Add repository
- `GET /api/repositories/{id}` - Get repository
- `DELETE /api/repositories/{id}` - Delete repository
- `POST /api/repositories/batch` - Batch add repositories
- `POST /api/repositories/{id}/sync` - Sync repository data
- `GET /api/repositories/date-ranges` - Get date ranges for repositories

### GitHub
- `GET /api/github/me` - Get authenticated GitHub user
- `GET /api/github/owners/{owner}/repos` - List repositories by owner

### Metrics
- `GET /api/metrics/cycle-time` - Cycle time analysis
- `GET /api/metrics/reviews` - Review analysis
- `GET /api/metrics/dora` - DORA metrics
- `GET /api/metrics/productivity-score` - Productivity score
- `GET /api/metrics/daily` - Daily aggregated metrics
- `GET /api/metrics/pull-requests` - Pull request list

### Sprints
- `GET /api/sprints` - List sprints
- `POST /api/sprints` - Create sprint
- `GET /api/sprints/{id}` - Get sprint
- `GET /api/sprints/{id}/performance` - Sprint performance

### Bot Users
- `GET /api/bot-users` - List bot users
- `POST /api/bot-users` - Add bot user
- `DELETE /api/bot-users` - Delete bot user

### Team
- `GET /api/team/members` - List team members
- `GET /api/team/members/{id}/stats` - Member statistics
- `GET /api/team/members/{id}/pull-requests` - Member pull requests
- `GET /api/team/members/{id}/reviews` - Member reviews

### Job
- `PUT /api/job/sync` - Trigger data sync job

## Project Structure

```
dora-yaki/
├── backend/
│   ├── cmd/httpserver/           # Entry point
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handler/         # HTTP handlers (metrics, repository, team, job, etc.)
│   │   │   ├── middleware/       # CORS, logger, cache
│   │   │   └── router.go        # Route definitions
│   │   ├── config/              # Configuration
│   │   ├── datastore/           # Cloud Datastore client
│   │   ├── domain/model/        # Domain models
│   │   ├── github/              # GitHub API client & collector
│   │   └── metrics/             # Calculator & aggregator
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── api/             # API client
│   │   │   ├── components/      # Svelte components (incl. PeriodSelector)
│   │   │   ├── i18n/            # Internationalization
│   │   │   ├── stores/          # Svelte stores
│   │   │   └── utils/           # Utility functions
│   │   └── routes/              # SvelteKit pages
│   ├── Dockerfile
│   └── package.json
├── terraform/                   # Infrastructure as Code
├── docs/
│   └── ARCHITECTURE.md
├── compose.yml
└── README.md
```

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `GITHUB_TOKEN` | GitHub Personal Access Token | Yes |
| `GCP_PROJECT_ID` | Google Cloud Project ID | Yes (development) |
| `PORT` | Backend server port (default: 7202) | No |
| `ENVIRONMENT` | development / production | No |
| `TZ_OFFSET` | Timezone offset (e.g. `+09:00`, `-05:30`). Defaults to UTC | No |
| `VITE_API_BASE` | Backend API base path (frontend, default: `/api`) | No |

## Infrastructure (Terraform)

GCP resources are managed with Terraform (`terraform/` directory).

### Managed Resources

| Resource | Description |
|----------|-------------|
| GCP APIs | Cloud Functions, Cloud Run, Cloud Build, Cloud Scheduler (periodic GitHub data sync), Secret Manager |
| Service Account | `dora-yaki-api` (Datastore access, Secret Manager access) |
| Secret Manager | `GITHUB_TOKEN` secret shell |
| Datastore Indexes | Composite indexes for PullRequest, Review, Deployment, DailyMetrics, Sprint |

### Setup

1. Install [Terraform](https://developer.hashicorp.com/terraform/install) (>= 1.0)

2. Configure variables:
```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your GCP project ID and region
```

| Variable | Description | Default |
|----------|-------------|---------|
| `project_id` | GCP Project ID | (required) |
| `region` | GCP Region | `asia-northeast1` |

3. Initialize and apply:
```bash
terraform init
terraform plan    # Review changes
terraform apply   # Apply changes
```

4. Add the GitHub token value to the created secret:
```bash
echo -n "your_github_token" | gcloud secrets versions add GITHUB_TOKEN --data-file=-
```

> **Note**: Terraform creates the secret definition only (without a value). The actual token value must be set via `gcloud` or the GCP Console.

## Security Considerations

> **Important**: This application does **not** include built-in authentication or authorization. All API endpoints are publicly accessible by default. When deploying to production, you **must** protect both the frontend and backend with an external authentication layer.

Recommended approaches:

- [Cloud IAP (Identity-Aware Proxy)](https://cloud.google.com/iap) - For GCP deployments
- [Cloudflare Access](https://www.cloudflare.com/products/zero-trust/access/) - For Cloudflare-based deployments
- [Google Cloud Armor (WAF)](https://cloud.google.com/armor) - Web Application Firewall for GCP
- OAuth2 Proxy / reverse proxy with authentication

> **Note**: The default CORS configuration allows all origins (`*`) for ease of local development. In production, restrict allowed origins to your frontend domain by modifying `backend/internal/api/router.go`.

## Deployment

### Backend (Cloud Functions gen2)

Deploy backend:
```bash
cd backend
make deploy
```

### Frontend

Frontend can be deployed to one of the following:

- **Cloud Run**: `cd frontend && make deploy-cloudrun`
- **Cloudflare Pages**: `cd frontend && make deploy-cloudflare`
- **Firebase Hosting**: `cd frontend && make deploy-firebase`

### Cloud Scheduler (optional)

Set up a periodic sync job to automatically fetch GitHub data:

```bash
cd backend
make schedule-create project=your-gcp-project region=asia-northeast1
```

To update the schedule:
```bash
make schedule-update project=your-gcp-project region=asia-northeast1
```

## Vibe Coding

This project was built almost entirely through **Vibe Coding** — AI-assisted development using [Claude Code](https://claude.com/claude-code).

- Architecture design, backend/frontend implementation, Terraform IaC, and documentation were all generated through AI pair programming
- Human role: requirements definition, design decisions, code review, and final approval
- AI role: code generation, refactoring, testing, and documentation

## License

This project is licensed under the [GNU Affero General Public License v3.0 (AGPL-3.0)](./LICENSE).

Per **AGPL-3.0 Section 13**, if you modify this software and make it available over a network, you must provide access to the corresponding source code to all users interacting with it remotely. The application includes a source code link in the UI footer to comply with this requirement.
