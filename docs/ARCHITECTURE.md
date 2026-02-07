# DORA-yaki Dashboard - Architecture

## Overview

A dashboard application that periodically collects GitHub metrics to visualize and analyze development productivity.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.25, net/http (stdlib) |
| Frontend | SvelteKit 2 (Svelte 5), Chart.js, date-fns, pnpm |
| Database | Google Cloud Datastore |
| Infrastructure | Google Cloud Functions / Cloud Run, Terraform |
| Container | Distroless (debian12) production images |
| External API | GitHub REST API |

## Features

### 1. DORA Metrics

Measures the four key DORA (DevOps Research and Assessment) metrics:

| Metric | Description | Data Source |
|--------|-------------|-------------|
| Deployment Frequency | How often deployments occur | GitHub Releases/Tags |
| Lead Time for Changes | Time from commit to production | PR merged_at - first_commit_at |
| Change Failure Rate | Percentage of failed deployments | Issue labels (bug, hotfix) |
| MTTR | Mean time to recover from failure | Issue open -> close time |

### 2. Cycle Time Analysis

Detailed analysis of the PR lifecycle:

```
[First Commit] -> [PR Open] -> [First Review] -> [Approved] -> [Merged]
     |                |              |               |            |
     +-- Coding ------+-- Pickup ----+-- Review -----+-- Merge ---+
```

- **Coding Time**: First commit to PR open
- **Pickup Time**: PR open to first review
- **Review Time**: First review to approval
- **Merge Time**: Approval to merge
- **Total Cycle Time**: End-to-end duration

### 3. Review Analysis

Analyzes code review efficiency and quality:

- Review and comment counts (per PR, per reviewer)
- Per-reviewer statistics
- Review response time
- Approve / Request Changes ratio

### 4. Productivity Score

Composite score (0-100) integrating all metrics:

```
Score = w1 * CycleTimeScore
      + w2 * ReviewScore
      + w3 * DeploymentScore
      + w4 * QualityScore
```

- Team-wide and individual scores
- Time-series trends
- Improvement recommendations

### 5. Team Analytics

Per-member statistics with detailed views:

- Daily and weekly activity charts
- PR creation and merge history
- Review activity
- Code change statistics by file extension

### 6. Bot User Management

Exclude bot accounts (e.g., dependabot, renovate) from metrics calculations.

### 7. Multi-language Support (i18n)

Lightweight i18n built on Svelte writable/derived stores (no external library):

- 8 languages: ja (default), en, zh-TW, zh-CN, ko, es, fr, de
- Browser language auto-detection
- localStorage persistence
- Fallback chain: current locale -> ja -> key name
- String interpolation with `{param}` syntax

## Architecture Diagram

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

## Backend Architecture

### Layers

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

## Frontend Architecture

### Pages

| Route | Description |
|-------|-------------|
| `/` | Dashboard with productivity score, key metrics, DORA, charts |
| `/metrics` | Detailed metrics analysis with cycle time, reviews, DORA |
| `/repo` | Repository list with summary metrics |
| `/repo/[id]` | Single repository detail view |
| `/team` | Team members list with summary stats |
| `/team/[id]` | Individual member detail with daily/weekly charts |
| `/bots` | Bot user management |
| `/repositories` | Repository management (add, sync, delete) |

### Components

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

### State Management

- `repositories.ts` - Repository selection state (`writable<string[]>`, empty = all repos)
- `metrics.ts` - Metrics utilities (formatting, trend helpers)
- `flash.ts` - Flash message store

### i18n

- `i18n/index.ts` - Locale store, `t` derived store, browser detection, localStorage persistence
- `i18n/translations.ts` - Translation dictionary (~170 keys x 8 languages)
- Template usage: `{$t('key')}` or `{$t('key', { param: value })}`
- Chart.js usage: `get(t)('key')` inside `onMount` / callback contexts

## Data Models

### Repository

```go
type Repository struct {
    ID             string     `datastore:"id"`
    Owner          string     `datastore:"owner"`
    Name           string     `datastore:"name"`
    FullName       string     `datastore:"full_name"`
    Private        bool       `datastore:"private"`
    CreatedAt      time.Time  `datastore:"created_at"`
    UpdatedAt      time.Time  `datastore:"updated_at"`
    LastSyncedAt   *time.Time `datastore:"last_synced_at"`
    ProcessStartAt *time.Time `datastore:"process_start_at"`
}
```

### PullRequest

```go
type PullRequest struct {
    ID            string         `datastore:"id"`
    RepositoryID  string         `datastore:"repository_id"`
    Number        int            `datastore:"number"`
    Title         string         `datastore:"title,noindex"`
    Author        string         `datastore:"author"`
    State         string         `datastore:"state"`
    Draft         bool           `datastore:"draft"`
    CreatedAt     time.Time      `datastore:"created_at"`
    UpdatedAt     time.Time      `datastore:"updated_at"`
    MergedAt      *time.Time     `datastore:"merged_at"`
    ClosedAt      *time.Time     `datastore:"closed_at"`
    FirstCommitAt *time.Time     `datastore:"first_commit_at"`
    FirstReviewAt *time.Time     `datastore:"first_review_at"`
    ApprovedAt    *time.Time     `datastore:"approved_at"`
    Additions     int            `datastore:"additions"`
    Deletions     int            `datastore:"deletions"`
    ChangedFiles  int            `datastore:"changed_files"`
    CommitCount   int            `datastore:"commit_count"`
    FileExtStats  []FileExtStats `datastore:"file_ext_stats,flatten"`
}
```

### Review

```go
type Review struct {
    ID            string    `datastore:"id"`
    PullRequestID string    `datastore:"pull_request_id"`
    RepositoryID  string    `datastore:"repository_id"`
    Reviewer      string    `datastore:"reviewer"`
    State         string    `datastore:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
    Body          string    `datastore:"body,noindex"`
    SubmittedAt   time.Time `datastore:"submitted_at"`
    CommentsCount int       `datastore:"comments_count"`
}
```

### DailyMetrics (Aggregated)

```go
type DailyMetrics struct {
    ID           string    `datastore:"id"` // repository_id:date
    RepositoryID string    `datastore:"repository_id"`
    Date         time.Time `datastore:"date"`

    // Cycle Time (in hours)
    AvgCycleTime  float64 `datastore:"avg_cycle_time"`
    AvgCodingTime float64 `datastore:"avg_coding_time"`
    AvgPickupTime float64 `datastore:"avg_pickup_time"`
    AvgReviewTime float64 `datastore:"avg_review_time"`
    AvgMergeTime  float64 `datastore:"avg_merge_time"`

    // PR Metrics
    PRsOpened int `datastore:"prs_opened"`
    PRsMerged int `datastore:"prs_merged"`
    PRsClosed int `datastore:"prs_closed"`

    // Review Metrics
    ReviewsSubmitted int     `datastore:"reviews_submitted"`
    AvgReviewsPerPR  float64 `datastore:"avg_reviews_per_pr"`

    // Code Metrics
    TotalAdditions int `datastore:"total_additions"`
    TotalDeletions int `datastore:"total_deletions"`

    // DORA Metrics
    DeploymentCount   int     `datastore:"deployment_count"`
    ChangeFailureRate float64 `datastore:"change_failure_rate"`

    // Contributors
    ActiveContributors int `datastore:"active_contributors"`
}
```

## API Endpoints

### Repositories

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/repositories` | List repositories |
| POST | `/api/repositories` | Add repository |
| GET | `/api/repositories/{id}` | Get repository |
| DELETE | `/api/repositories/{id}` | Delete repository |
| POST | `/api/repositories/batch` | Batch add repositories |
| POST | `/api/repositories/{id}/sync` | Sync repository data |
| GET | `/api/repositories/date-ranges` | Get date ranges for repositories |

### GitHub

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/github/me` | Get authenticated user |
| GET | `/api/github/owners/{owner}/repos` | List owner's repositories |

### Metrics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/metrics/cycle-time` | Cycle time analysis |
| GET | `/api/metrics/reviews` | Review analysis |
| GET | `/api/metrics/dora` | DORA metrics |
| GET | `/api/metrics/productivity-score` | Productivity score |
| GET | `/api/metrics/daily` | Daily aggregated metrics |
| GET | `/api/metrics/pull-requests` | Pull request list |

### Sprints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/sprints` | List sprints |
| POST | `/api/sprints` | Create sprint |
| GET | `/api/sprints/{id}` | Get sprint |
| GET | `/api/sprints/{id}/performance` | Sprint performance |

### Bot Users

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/bot-users` | List bot users |
| POST | `/api/bot-users` | Add bot user |
| DELETE | `/api/bot-users` | Delete bot user |

### Team

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/team/members` | List team members |
| GET | `/api/team/members/{id}/stats` | Member statistics |
| GET | `/api/team/members/{id}/pull-requests` | Member pull requests |
| GET | `/api/team/members/{id}/reviews` | Member reviews |

### Job

| Method | Path | Description |
|--------|------|-------------|
| PUT | `/api/job/sync` | Trigger data sync job |

## Container Images

### Backend

```dockerfile
# Build: golang:1.25-bookworm
# Runtime: gcr.io/distroless/base-debian12
```

Static Go binary (`CGO_ENABLED=0`) runs directly on distroless base image.

### Frontend

```dockerfile
# Build: node:24-bookworm (pnpm via corepack)
# Runtime: gcr.io/distroless/nodejs24-debian12
```

SvelteKit adapter-node output runs on distroless Node.js image.

## Directory Structure

```
dora-yaki/
├── backend/
│   ├── cmd/
│   │   └── httpserver/
│   │       └── main.go
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handler/
│   │   │   │   ├── bot_user.go
│   │   │   │   ├── github.go
│   │   │   │   ├── metrics.go
│   │   │   │   ├── repository.go
│   │   │   │   ├── sprint.go
│   │   │   │   ├── team.go
│   │   │   │   └── job.go
│   │   │   ├── middleware/
│   │   │   │   ├── cache.go
│   │   │   │   └── middleware.go
│   │   │   └── router.go
│   │   ├── config/
│   │   │   └── config.go
│   │   ├── datastore/
│   │   │   └── client.go
│   │   ├── domain/
│   │   │   └── model/
│   │   │       ├── bot.go
│   │   │       ├── metrics.go
│   │   │       └── models.go
│   │   ├── github/
│   │   │   ├── client.go
│   │   │   └── collector.go
│   │   └── metrics/
│   │       ├── aggregator.go
│   │       └── calculator.go
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── app.css
│   │   ├── app.html
│   │   ├── lib/
│   │   │   ├── api/
│   │   │   │   └── client.ts
│   │   │   ├── components/
│   │   │   │   ├── CycleTimeChart.svelte
│   │   │   │   ├── FlashMessage.svelte
│   │   │   │   ├── MemberDailyChart.svelte
│   │   │   │   ├── MemberWeeklyChart.svelte
│   │   │   │   ├── MetricCard.svelte
│   │   │   │   ├── PeriodSelector.svelte
│   │   │   │   ├── PRActivityChart.svelte
│   │   │   │   └── ScoreGauge.svelte
│   │   │   ├── i18n/
│   │   │   │   ├── index.ts
│   │   │   │   └── translations.ts
│   │   │   ├── stores/
│   │   │   │   ├── flash.ts
│   │   │   │   ├── metrics.ts
│   │   │   │   └── repositories.ts
│   │   │   └── utils/
│   │   └── routes/
│   │       ├── +layout.svelte
│   │       ├── +page.svelte
│   │       ├── bots/
│   │       ├── metrics/
│   │       ├── repo/
│   │       │   └── [id]/
│   │       ├── repositories/
│   │       └── team/
│   │           └── [id]/
│   ├── Dockerfile
│   ├── package.json
│   ├── pnpm-lock.yaml
│   ├── svelte.config.js
│   ├── tsconfig.json
│   └── vite.config.ts
├── terraform/
│   ├── apis.tf
│   ├── datastore.tf
│   ├── iam.tf
│   ├── main.tf
│   ├── variables.tf
│   └── terraform.tfvars.example
├── docs/
│   └── ARCHITECTURE.md
├── compose.yml
└── README.md
```

## Environment Variables

### Backend

| Variable | Description | Required |
|----------|-------------|----------|
| `GITHUB_TOKEN` | GitHub Personal Access Token | Yes |
| `GCP_PROJECT_ID` | Google Cloud Project ID | Yes |
| `PORT` | Server port (default: 7202) | No |
| `ENVIRONMENT` | development / production | No |
| `TZ_OFFSET` | Timezone offset (e.g. `+09:00`, `-05:30`). Defaults to UTC | No |
| `SYNC_INTERVAL_MINUTES` | Sync interval in minutes (default: 60) | No |
| `SYNC_LOCK_TTL_MINUTES` | Sync lock TTL in minutes (default: 10) | No |

### Frontend

| Variable | Description | Required |
|----------|-------------|----------|
| `VITE_API_BASE` | Backend API base path (default: `/api`) | No |
| `VITE_DEFAULT_LOCALE` | Default locale: ja, en, zh-TW, zh-CN, ko, es, fr, de (default: ja) | No |

## Deployment

### Backend (Cloud Functions gen2)

Backend is deployed as a Cloud Functions gen2 function:

- `gcloud functions deploy` via Makefile (`make deploy`)

### Frontend

Frontend can be deployed to one of:

1. **Cloud Run**: `make deploy-cloudrun`
2. **Cloudflare Pages**: `make deploy-cloudflare`
3. **Firebase Hosting**: `make deploy-firebase`

### Scheduled Data Collection

Cloud Scheduler triggers `PUT /api/job/sync` every 5 minutes. Each invocation syncs one repository at a time (the one with the oldest `LastSyncedAt` that exceeds the configured interval):

- Collects pull requests, reviews, and deployments from GitHub
- Aggregates daily metrics
- Uses Datastore-based distributed locking to prevent concurrent execution
