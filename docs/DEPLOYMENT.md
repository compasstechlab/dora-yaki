# Deployment Guide

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) (>= 1.0)
- Google Cloud Project
- GitHub Personal Access Token (see [GitHub Token Setup](#github-token-setup))

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

1. Configure variables:
```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your GCP project ID and region
```

| Variable | Description | Default |
|----------|-------------|---------|
| `project_id` | GCP Project ID | (required) |
| `region` | GCP Region | `asia-northeast1` |

2. Initialize and apply:
```bash
terraform init
terraform plan    # Review changes
terraform apply   # Apply changes
```

3. Add the GitHub token value to the created secret:
```bash
echo -n "your_github_token" | gcloud secrets versions add GITHUB_TOKEN --data-file=-
```

> **Note**: Terraform creates the secret definition only (without a value). The actual token value must be set via `gcloud` or the GCP Console.

## Backend (Cloud Functions gen2)

```bash
cd backend
make deploy
```

## Frontend

Frontend can be deployed to one of the following:

- **Cloud Run**: `cd frontend && make deploy-cloudrun`
- **Cloudflare Pages**: `cd frontend && make deploy-cloudflare`
- **Firebase Hosting**: `cd frontend && make deploy-firebase`

## Cloud Scheduler (optional)

Set up a periodic sync job to automatically fetch GitHub data:

```bash
cd backend
make schedule-create project=your-gcp-project region=asia-northeast1
```

To update the schedule:
```bash
make schedule-update project=your-gcp-project region=asia-northeast1
```

## Security Considerations

> **Important**: This application does **not** include built-in authentication or authorization. All API endpoints are publicly accessible by default. When deploying to production, you **must** protect both the frontend and backend with an external authentication layer.

Recommended approaches:

- [Cloud IAP (Identity-Aware Proxy)](https://cloud.google.com/iap) - For GCP deployments
- [Cloudflare Access](https://www.cloudflare.com/products/zero-trust/access/) - For Cloudflare-based deployments
- [Google Cloud Armor (WAF)](https://cloud.google.com/armor) - Web Application Firewall for GCP
- OAuth2 Proxy / reverse proxy with authentication

> **Note**: The default CORS configuration allows all origins (`*`) for ease of local development. In production, restrict allowed origins to your frontend domain by modifying `backend/internal/api/router.go`.

## GitHub Token Setup

DORA-yaki requires a GitHub Personal Access Token to fetch repository data, pull requests, reviews, and deployments.

### Option 1: Fine-grained Personal Access Token (Recommended)

Fine-grained PATs provide more granular permission control and are recommended by GitHub.

1. Go to [GitHub Settings > Developer settings > Fine-grained tokens](https://github.com/settings/personal-access-tokens/new)
2. Set **Token name** and **Expiration**
3. Under **Resource owner**, select the organization you want to access (e.g. `your-org`)
   - If the organization is not listed, an org admin must enable fine-grained PAT access in [Organization settings > Personal access tokens](https://github.com/organizations/YOUR_ORG/settings/personal-access-tokens)
4. Under **Repository access**, select **All repositories** or choose specific repositories
5. Under **Permissions > Repository permissions**, grant the following:

| Permission | Access | Purpose |
|-----------|--------|---------|
| **Metadata** | Read | List repositories |
| **Contents** | Read | Read repository data |
| **Pull requests** | Read | Fetch PR data for cycle time / review metrics |
| **Deployments** | Read | Fetch deployment data for DORA deployment frequency |

6. Click **Generate token** and copy it to `.env`

### Option 2: Classic Personal Access Token

1. Go to [GitHub Settings > Developer settings > Tokens (classic)](https://github.com/settings/tokens/new)
2. Select the following scopes:

| Scope | Purpose |
|-------|---------|
| `repo` | Full access to repositories (includes PRs, deployments) |
| `read:org` | Read organization membership and repository listing |

3. Click **Generate token** and copy it to `.env`

> **Note**: For organization repositories, the token owner must be a member of the organization. If using a fine-grained PAT, the organization admin must allow fine-grained PAT access.
