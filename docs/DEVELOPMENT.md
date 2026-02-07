# Development Guide

## Prerequisites

- Go 1.25+
- Node.js 24+
- pnpm
- GitHub Personal Access Token (see [Deployment Guide](./DEPLOYMENT.md#github-token-setup))

## Manual Setup (without Docker)

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

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `GITHUB_TOKEN` | GitHub Personal Access Token | Yes |
| `GCP_PROJECT_ID` | Google Cloud Project ID | Yes (development) |
| `PORT` | Backend server port (default: 7202) | No |
| `ENVIRONMENT` | development / production | No |
| `TZ_OFFSET` | Timezone offset (e.g. `+09:00`, `-05:30`). Defaults to UTC | No |
| `FUNCTION_TARGET` | Cloud Functions entry point (default: `RunHTTPServer`) | No |
| `API_BACKEND` | Backend API URL for server-side proxy (default: `http://localhost:7202`) | No |
| `VITE_API_BASE` | Backend API base path (frontend, default: `/api`) | No |
| `VITE_DEFAULT_LOCALE` | Default locale (default: `ja`) | No |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Cloud Run                                │
│  ┌─────────────────────┐    ┌─────────────────────────────────┐ │
│  │   SvelteKit App     │    │        Go Backend               │ │
│  │   (Frontend)        │    │                                 │ │
│  │  ┌───────────────┐  │    │  ┌───────────┐ ┌─────────────┐ │ │
│  │  │  Dashboard    │  │    │  │   REST    │ │   GitHub    │ │ │
│  │  │  Components   │──┼────┼──│   API     │ │   Collector │ │ │
│  │  └───────────────┘  │    │  └─────┬─────┘ └──────┬──────┘ │ │
│  │  ┌───────────────┐  │    │        │              │        │ │
│  │  │  Charts       │  │    │  ┌─────▼──────────────▼──────┐ │ │
│  │  │  (Chart.js)   │  │    │  │     Middleware Layer      │ │ │
│  │  └───────────────┘  │    │  │  (CORS, Logger, Cache)    │ │ │
│  │  ┌───────────────┐  │    │  └─────┬──────────────┬──────┘ │ │
│  │  │  i18n         │  │    │        │              │        │ │
│  │  │  (8 langs)    │  │    │  ┌─────▼─────┐  ┌─────▼─────┐ │ │
│  │  └───────────────┘  │    │  │ Datastore │  │  GitHub   │ │ │
│  └─────────────────────┘    │  │  Client   │  │  Client   │ │ │
│                             │  └─────┬─────┘  └─────┬─────┘ │ │
│                             └────────┼──────────────┼───────┘ │
└──────────────────────────────────────┼──────────────┼─────────┘
                                       │              │
                              ┌────────▼────────┐ ┌───▼────────┐
                              │ Cloud Datastore │ │ GitHub API │
                              └─────────────────┘ └────────────┘
```

### Backend Layers

```
cmd/httpserver/main.go      Entry point, server startup
    │
    ▼
api/router.go               Route definitions (net/http)
    │
    ▼
api/middleware/              Cross-cutting concerns
├── middleware.go            CORS, Logger, Recovery, RequestID
└── cache.go                Response cache (50-min TTL)
    │
    ▼
api/handler/                HTTP handlers (request/response)
├── metrics.go              Cycle time, reviews, DORA, productivity
├── repository.go           Repository CRUD & sync
├── team.go                 Team member stats
├── bot_user.go             Bot user management
├── github.go               GitHub API proxy
├── sprint.go               Sprint management
└── job.go                  Data sync job handler
    │
    ▼
metrics/                    Business logic
├── calculator.go           Metrics computation
└── aggregator.go           Data aggregation
    │
    ▼
datastore/client.go         Cloud Datastore persistence
github/                     GitHub data collection
├── client.go               API client wrapper
└── collector.go            Data sync orchestration
```

### Key Design Decisions

- **Handler-level aggregation**: Multi-repo aggregation is done at the handler level via loops, not in the calculator/aggregator layers
- **Datastore methods are per-repository**: Each method operates on a single repo; cross-repo queries are composed at the handler level
- **Response caching**: 50-minute TTL with in-memory cache to reduce Datastore reads
- **Static binary**: `CGO_ENABLED=0` for distroless compatibility

### Cycle Time Breakdown

```
[First Commit] -> [PR Open] -> [First Review] -> [Approved] -> [Merged]
     |                |              |               |            |
     +-- Coding ------+-- Pickup ----+-- Review -----+-- Merge ---+
```

### Frontend Key Components

| Component | Description |
|-----------|-------------|
| `CycleTimeChart` | Line chart for cycle time trends |
| `PRActivityChart` | Bar chart for PR open/merge/review activity |
| `MemberDailyChart` | Daily activity heatmap for individual members |
| `MemberWeeklyChart` | Weekly activity chart for individual members |
| `MetricCard` | Reusable metric display card |
| `ScoreGauge` | Circular gauge for productivity score |
| `PeriodSelector` | Date period selection component |
| `FlashMessage` | Toast notification component |

### i18n

Lightweight i18n built on Svelte writable/derived stores (no external library):

- 8 languages: ja (default), en, zh-TW, zh-CN, ko, es, fr, de
- Browser language auto-detection with localStorage persistence
- Fallback chain: current locale -> ja -> key name
- Template usage: `{$t('key')}` or `{$t('key', { param: value })}`

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
│   │   ├── metrics/             # Calculator & aggregator
│   │   └── timeutil/            # Timezone offset handling
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
│   ├── DEPLOYMENT.md
│   └── DEVELOPMENT.md
├── .env.example                 # Environment variables template
├── compose.yml
└── README.md
```
