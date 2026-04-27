# Deployment Guide

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) (>= 1.0)
- Google Cloud Project
- A GitHub OAuth App (see [GitHub OAuth App Setup](#github-oauth-app-setup))

## GitHub OAuth App Setup

dora-yaki authenticates every user via GitHub OAuth. There is no longer a
shared GitHub Personal Access Token.

1. Open <https://github.com/settings/developers> -> **OAuth Apps** -> **New OAuth App**.
2. Fill in:
    - **Application name**: e.g. `dora-yaki (production)`
    - **Homepage URL**: the frontend origin, e.g. `https://dora-yaki.example.com`
    - **Authorization callback URL**: `https://<backend-domain>/api/auth/github/callback`
3. Click **Register application**.
4. Copy the **Client ID** and click **Generate a new client secret**.
5. (Recommended) In the OAuth App settings, enable **Expire user authorization tokens** so refresh tokens are issued.

The OAuth flow requests `read:user`, `user:email`, and `repo` scopes. The
`repo` scope is required because the backend reads private repository
metadata that the calling user is permitted to see on GitHub.

> Each user logs in with their own GitHub account. dora-yaki stores their
> OAuth tokens **encrypted** in Datastore (AES-256-GCM or Cloud KMS) and
> rotates between users when one user's token fails on a private repo.

## Infrastructure (Terraform)

GCP resources are managed with Terraform (`terraform/` directory).

### Managed Resources

| Resource | Description |
|----------|-------------|
| GCP APIs | Cloud Functions, Cloud Run, Cloud Build, Cloud Scheduler, Secret Manager, Cloud KMS, Firestore |
| Service Account | `dora-yaki-api` (Datastore access, Secret Manager access, optional KMS encrypter/decrypter, Cloud Run invoker) |
| Secret Manager | `GITHUB_OAUTH_CLIENT_ID`, `GITHUB_OAUTH_CLIENT_SECRET`, `AUTH_JWT_SECRET`, `ENCRYPTION_KEY_BASE64`, `JOB_AUTH_KEY` |
| Cloud KMS (optional) | Key ring `dora-yaki`, key `tokens` (90-day rotation). Provisioned only when `use_kms = true` |
| Cloud Scheduler | Terraform-managed sync and permission-check jobs |
| Datastore Indexes | Composite indexes for PullRequest, Review, Deployment, DailyMetrics, Sprint, RepositoryAccess |

### Setup

1. Configure variables:
```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your GCP project ID, region, resource names,
# and optionally backend_service_url + use_kms.
```

| Variable | Description | Default |
|----------|-------------|---------|
| `project_id` | GCP Project ID | (required) |
| `region` | GCP Region | `asia-northeast1` |
| `resource_name_prefix` | Prefix for generated resource names such as scheduler jobs | `dora-yaki` |
| `service_account_id` | Backend service account ID | `dora-yaki-api` |
| `service_account_display_name` | Backend service account display name | `DORA-Yaki API` |
| `secret_replication_locations` | User-managed Secret Manager replication locations. Empty means automatic replication | `[]` |
| `create_datastore_database` | Create the default Firestore database in Datastore mode | `true` |
| `datastore_location_id` | Datastore mode database location. Empty means `region` | `""` |
| `backend_service_url` | Backend base URL used by Cloud Scheduler. Set after the first deploy. | `""` (skips scheduler) |
| `scheduler_auth_key` | Shared key sent by Cloud Scheduler in `X-DORA-YAKI-JOB-KEY`. Must match `JOB_AUTH_KEY` | `""` (skips scheduler) |
| `use_kms` | Provision a Cloud KMS key for token encryption | `false` |
| `kms_location` | KMS key ring location | `asia-northeast1` |
| `kms_key_ring_name` | KMS key ring name | `dora-yaki` |
| `kms_crypto_key_name` | KMS crypto key name | `tokens` |
| `sync_job_name` | Cloud Scheduler sync job name. Empty means `<resource_name_prefix>-sync` | `""` |
| `sync_schedule` | Cron for repository sync | `*/15 * * * *` |
| `permission_check_job_name` | Cloud Scheduler permission-check job name. Empty means `<resource_name_prefix>-permission-check` | `""` |
| `permission_check_schedule` | Cron for daily ACL refresh | `0 3 * * *` |
| `scheduler_time_zone` | IANA TZ for Cloud Scheduler | `Asia/Tokyo` |

2. Initialize and apply:
```bash
terraform init
terraform plan    # Review changes
terraform apply   # Apply changes
```

3. Generate and store the runtime secrets:

```bash
# Random secrets
JWT_SECRET=$(openssl rand -base64 32)
ENC_KEY=$(openssl rand -base64 32)
JOB_KEY=$(openssl rand -base64 32)

# Push them into Secret Manager (Terraform created the secret shells; you add the values).
gcloud secrets versions add GITHUB_OAUTH_CLIENT_ID    --data-file=- <<<"<from GitHub OAuth App>"
gcloud secrets versions add GITHUB_OAUTH_CLIENT_SECRET --data-file=- <<<"<from GitHub OAuth App>"
gcloud secrets versions add AUTH_JWT_SECRET           --data-file=- <<<"$JWT_SECRET"
gcloud secrets versions add ENCRYPTION_KEY_BASE64     --data-file=- <<<"$ENC_KEY"
gcloud secrets versions add JOB_AUTH_KEY              --data-file=- <<<"$JOB_KEY"
```

> **Note**: Terraform only creates the secret shells. The actual values must be set via `gcloud` or the GCP Console. Treat all five secrets as sensitive.
> Secret IDs intentionally match the backend environment variable names because
> `backend/Makefile` uses fixed `--set-secrets` bindings.

> **Cloud KMS alternative**: if you set `use_kms = true`, skip
> `ENCRYPTION_KEY_BASE64` and instead set `ENCRYPTION_KMS_KEY` on the
> backend runtime to the resource path of the provisioned key
> (`terraform output -raw kms_crypto_key_id`, or
> `projects/<project>/locations/<location>/keyRings/<kms_key_ring_name>/cryptoKeys/<kms_crypto_key_name>`).
> `backend/Makefile` automatically maps `BACKEND_ENCRYPTION_KMS_KEY` to the
> runtime `ENCRYPTION_KMS_KEY` when that deploy variable is set; otherwise it
> binds the `ENCRYPTION_KEY_BASE64` Secret Manager secret. KMS takes priority
> over the base64 key when both are set at runtime.

## Backend (Cloud Functions gen2)

Configure the runtime environment variables on the function (or Cloud Run service):

| Variable | Source | Notes |
|----------|--------|-------|
| `GITHUB_OAUTH_CLIENT_ID` | Secret Manager | |
| `GITHUB_OAUTH_CLIENT_SECRET` | Secret Manager | |
| `AUTH_JWT_SECRET` | Secret Manager | |
| `ENCRYPTION_KEY_BASE64` | Secret Manager | omit when using KMS |
| `JOB_AUTH_KEY` | Secret Manager | Required in production. Shared key for `/api/job/*` endpoints |
| `ENCRYPTION_KMS_KEY` | env (plain) | KMS resource path; omit when using base64 |
| `OAUTH_REDIRECT_URL` | env (plain) | `https://<backend-domain>/api/auth/github/callback` |
| `FRONTEND_URL` | env (plain) | Frontend origin (used for post-login redirect) |
| `AUTH_COOKIE_DOMAIN` | env (plain) | Optional; set when frontend & backend are on different subdomains of the same root |
| `GCP_PROJECT_ID` | env (plain) | |
| `TZ_OFFSET` | env (plain) | Optional |

Then deploy:

```bash
cd backend
make deploy
```

For KMS deployments, set `BACKEND_ENCRYPTION_KMS_KEY` before `make deploy`:

```bash
export BACKEND_ENCRYPTION_KMS_KEY="$(terraform -chdir=../terraform output -raw kms_crypto_key_id)"
make deploy
```

`backend/Makefile` reads `../.env` when the file exists. If it does not exist,
the Makefile uses variables from the current shell environment. When you
override Terraform names, set matching values in `.env` or exported
environment variables.
For example, Terraform `service_account_id` must match
`BACKEND_SERVICE_ACCOUNT_ID`. Secret Manager IDs are fixed to the backend
environment variable names used in `--set-secrets`.
Before deploying, the Makefile prints the resolved variables and asks for
confirmation with `Continue? [y/N]`.

After the first deploy, capture the service URL and set
`backend_service_url` and `scheduler_auth_key` in `terraform.tfvars`, then
`terraform apply` again to create the Cloud Scheduler jobs.
`scheduler_auth_key` must match the `JOB_AUTH_KEY` Secret Manager value.
Because Terraform sends this key as an HTTP header, it will be stored in
Terraform state; protect the state backend accordingly.

## Frontend

Frontend can be deployed to one of the following:

- **Cloud Run**: `cd frontend && make deploy-cloudrun`
- **Cloudflare Pages**: `cd frontend && make deploy-cloudflare`
- **Firebase Hosting**: `cd frontend && make deploy-firebase`

`frontend/Makefile` also reads `../.env` when the file exists, otherwise it
uses variables from the current shell environment. It validates the required
deployment variables before running. The frontend forwards `/api/*` to the backend;
configure `API_BACKEND` to the absolute backend URL for server-side proxying
when needed.
Frontend deploy targets also print the resolved variables and ask for
confirmation with `Continue? [y/N]`.

## Cloud Scheduler

Two jobs are managed by Terraform once `backend_service_url` and
`scheduler_auth_key` are set:

- `<resource_name_prefix>-sync` (default `*/15 * * * *`) - calls `PUT /api/job/sync`
- `<resource_name_prefix>-permission-check` (default `0 3 * * *`) - calls `PUT /api/job/permission-check`

Both send `X-DORA-YAKI-JOB-KEY` and an OIDC token issued for the configured
backend service account. The application validates the shared job key; the OIDC
token keeps Cloud Run / Cloud Functions invocation compatible with IAM checks.

## Security Considerations

User authentication is now built in via GitHub OAuth. Every `/api/*`
endpoint (except `/api/auth/*` and `/api/cache/invalidate`) requires a
session cookie issued by the OAuth callback.

Additional production hardening you should still consider:

- Set `FRONTEND_URL` to the exact frontend origin so CORS only allows that origin.
- Set and rotate `JOB_AUTH_KEY`; it protects scheduler/job endpoints.
- Set `AUTH_COOKIE_DOMAIN` so the session cookie is scoped tightly.
- Rotate `AUTH_JWT_SECRET` and `ENCRYPTION_KEY_BASE64` periodically (or use Cloud KMS with automatic 90-day rotation).
- Limit who can install the OAuth App by setting **Restrict to organization** in GitHub OAuth App settings.
- Front the backend with [Cloud Armor](https://cloud.google.com/armor) to mitigate abuse of the public OAuth endpoints.

## Migration from a Single GITHUB_TOKEN

Earlier versions of dora-yaki used a single shared `GITHUB_TOKEN` env var.
Migration:

1. Apply the new Terraform (creates the five secrets and KMS resources if enabled).
2. Generate `AUTH_JWT_SECRET` / `ENCRYPTION_KEY_BASE64` / `JOB_AUTH_KEY` and push them, plus the OAuth Client ID / Secret, to Secret Manager.
3. Redeploy the backend with the new env vars (`GITHUB_TOKEN` is no longer read). `backend/Makefile` uses `--set-secrets`, so legacy secret environment variables are replaced by the current list.
4. The first user who logs in becomes the seed for `RepositoryAccess`. Existing repository data remains intact; access is granted lazily via the daily permission-check job and on each user's first login.
5. Delete the old `GITHUB_TOKEN` secret once nothing references it.
