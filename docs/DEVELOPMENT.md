# Development Guide

## Prerequisites

- Go 1.25+
- Node.js 24+
- pnpm
- A GitHub OAuth App (see below)

## GitHub OAuth App Setup

dora-yaki uses GitHub OAuth for user login; every API call is gated behind a session cookie.
A standalone GitHub Personal Access Token is **not** required.

1. Open <https://github.com/settings/developers> -> **OAuth Apps** -> **New OAuth App**.
2. Fill in:
    - **Application name**: e.g. `dora-yaki (dev)`
    - **Homepage URL**: `http://localhost:7201`
    - **Authorization callback URL**: `http://localhost:7202/api/auth/github/callback`
3. Click **Register application**.
4. Copy the **Client ID** and click **Generate a new client secret** -> copy the secret.
5. (Recommended) Toggle **Expire user authorization tokens** in the OAuth App settings so refresh tokens are issued.
6. Set both values into your `.env`:
    ```bash
    GITHUB_OAUTH_CLIENT_ID=...
    GITHUB_OAUTH_CLIENT_SECRET=...
    ```

The OAuth flow asks for `read:user`, `user:email`, and `repo` scopes; the
`repo` scope is needed so the backend can read private repository metadata
that any logged-in user is permitted to see on GitHub.

## Manual Setup (without Docker)

Generate the runtime secrets once:

```bash
# AES-256 key used to encrypt OAuth tokens at rest
openssl rand -base64 32

# Session signing secret
openssl rand -base64 32
```

Copy `.env.example` to `.env` and fill in the values:

```bash
cp .env.example .env
$EDITOR .env
```

**Backend:**
```bash
cd backend
go mod download
set -a; source ../.env; set +a
go run ./cmd/httpserver/main.go
```

You should see `oauth login enabled` in the startup log. Without it the
backend exits with a configuration error.

**Frontend:**
```bash
cd frontend
pnpm install
pnpm run dev
```

Visit `http://localhost:7201/`. You will be redirected to `/login` ->
GitHub OAuth -> back to the app.

## Environment Variables

### Backend

| Variable | Description | Required |
|----------|-------------|----------|
| `GITHUB_OAUTH_CLIENT_ID` | GitHub OAuth App Client ID | Yes |
| `GITHUB_OAUTH_CLIENT_SECRET` | GitHub OAuth App Client Secret | Yes |
| `OAUTH_REDIRECT_URL` | Absolute callback URL (e.g. `http://localhost:7202/api/auth/github/callback`) | Yes |
| `AUTH_JWT_SECRET` | Random 32+ byte secret used to sign session cookies | Yes |
| `ENCRYPTION_KEY_BASE64` | base64-encoded 32-byte AES-256 key for token-at-rest encryption | Yes (or `ENCRYPTION_KMS_KEY`) |
| `ENCRYPTION_KMS_KEY` | Cloud KMS resource path (`projects/.../cryptoKeys/...`). Takes priority over `ENCRYPTION_KEY_BASE64` | No |
| `BACKEND_ENCRYPTION_KMS_KEY` | Deploy-time variable used by `backend/Makefile`; mapped to runtime `ENCRYPTION_KMS_KEY` | No |
| `FRONTEND_URL` | Frontend origin used for post-login redirects (e.g. `http://localhost:7201`) | Yes |
| `JOB_AUTH_KEY` | Shared key for `/api/job/*` endpoints. Required in production; optional in local development. If set, send it via `X-DORA-YAKI-JOB-KEY` or Basic auth username | Production only |
| `AUTH_COOKIE_DOMAIN` | Optional cookie domain (leave blank for localhost) | No |
| `GCP_PROJECT_ID` | Google Cloud Project ID | Yes |
| `PORT` | Backend server port (default: 7202) | No |
| `ENVIRONMENT` | development / production | No |
| `TZ_OFFSET` | Timezone offset (e.g. `+09:00`, `-05:30`). Defaults to UTC | No |
| `DATASTORE_EMULATOR_HOST` | Datastore emulator host (local dev only) | No |

### Frontend

| Variable | Description | Required |
|----------|-------------|----------|
| `API_BACKEND` | Backend URL used by the SvelteKit server-side proxy (default: `http://localhost:7202`) | No |
| `VITE_API_BASE` | Backend API base path (default: `/api`) | No |
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
├── auth.go                 GitHub OAuth login/callback/logout/me
├── metrics.go              Cycle time, reviews, DORA, productivity
├── repository.go           Repository CRUD & sync (filtered by RepositoryAccess)
├── team.go                 Team member stats
├── bot_user.go             Bot user management
├── github.go               GitHub API proxy (per-user)
├── sprint.go               Sprint management
├── job.go                  Repository sync job
└── permission_check.go     Daily user×repo permission refresh job
    │
    ▼
auth/                       Session cookies + GitHub OAuth flow
├── session.go              HMAC-SHA256 cookie sign/verify
├── cookie.go               Cookie helpers
├── middleware.go           RequireAuth gate
├── oauth_github.go         Authorize URL, code exchange, refresh
└── state.go                Signed OAuth `state` (CSRF + return_to)
    │
    ▼
crypto/                     Token-at-rest encryption
├── encryptor.go            Encryptor interface + factory (KMS > AES-256-GCM)
├── aesgcm.go               AES-256-GCM impl
└── kms.go                  Cloud KMS impl
    │
    ▼
metrics/                    Business logic
├── calculator.go           Metrics computation
└── aggregator.go           Data aggregation
    │
    ▼
datastore/                  Cloud Datastore persistence
├── client.go               Core kinds + lock
├── users.go                User entity
├── user_tokens.go          UserGitHubToken (encrypted access/refresh)
└── repository_access.go    RepositoryAccess (per-user ACL + token rotation pool)
github/                     GitHub data collection
├── client.go               API client wrapper
├── collector.go            Data sync orchestration
├── factory.go              Per-user client factory (auto-refresh)
├── rotation.go             Token rotation pool (LRU + invalid-on-401/403)
└── auth_error.go           IsAuthError helper
```

### Key Design Decisions

- **GitHub OAuth login required**: every `/api/*` request (except `/api/auth/*` and ops endpoints) is gated. There is no shared service token.
- **Per-user encrypted GitHub tokens**: `UserGitHubToken` stores AES-256-GCM (or KMS-encrypted) `access_token` + `refresh_token`; access tokens auto-refresh ~5 min before expiry.
- **Repository ACL with daily refresh**: a user's view of a repository is gated by `RepositoryAccess.CanAccess`, refreshed by `PUT /api/job/permission-check` (run daily).
- **Sync token rotation**: per-repo candidate pool ordered LRU; on 401/403 the offending user's token is marked invalid and the next candidate retries.
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

All `/api/*` endpoints except `/api/auth/*` and `/api/cache/invalidate`
require a valid session cookie issued by `GET /api/auth/github/callback`.

### Health
- `GET /health` - Health check

### Auth
- `GET /api/auth/github/login` - 302 redirect to GitHub OAuth authorize URL (accepts `?return_to=/path`)
- `GET /api/auth/github/callback` - OAuth callback; exchanges code, sets session cookie, redirects to frontend
- `POST /api/auth/logout` - Clear session cookie
- `GET /api/auth/me` - Return the logged-in user (`{id, login, name, avatarUrl}`) or 401

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
- `PUT /api/job/sync` - Trigger one-repo data sync (designed for Cloud Scheduler)
- `PUT /api/job/permission-check` - Refresh `RepositoryAccess` for every (user, repo); designed for daily Cloud Scheduler invocation

## Project Structure

```
dora-yaki/
├── backend/
│   ├── cmd/httpserver/           # Entry point
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handler/         # HTTP handlers (auth, metrics, repository, team, job, permission_check)
│   │   │   ├── middleware/       # CORS, logger, cache
│   │   │   └── router.go        # Route definitions
│   │   ├── auth/                # OAuth flow + session cookie + RequireAuth middleware
│   │   ├── config/              # Configuration
│   │   ├── crypto/              # Encryptor (AES-GCM / Cloud KMS)
│   │   ├── datastore/           # Cloud Datastore client
│   │   ├── domain/model/        # Domain models (incl. User, UserGitHubToken, RepositoryAccess)
│   │   ├── github/              # GitHub client + collector + per-user factory + rotation pool
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
