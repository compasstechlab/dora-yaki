# Service account, IAM and secret management

# Service account for dora-yaki API (Cloud Functions / Cloud Scheduler)
resource "google_service_account" "dora_yaki_api" {
  project      = var.project_id
  account_id   = "dora-yaki-api"
  display_name = "DORA-Yaki API"
  description  = "Service account for dora-yaki backend (Cloud Functions + Cloud Scheduler)"
}

# Datastore access
resource "google_project_iam_member" "dora_yaki_api_datastore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.dora_yaki_api.email}"
}

# Secret Manager secret definition (set the value via `gcloud secrets versions add` or Console)
resource "google_secret_manager_secret" "github_token" {
  project   = var.project_id
  secret_id = "GITHUB_TOKEN"

  replication {
    auto {}
    // if you want to use specified locations, use user_managed instead
    # user_managed {
    #   replicas {
    #     location = "asia-northeast1"
    #   }
    # }
  }

  depends_on = [google_project_service.secretmanager]
}

# Secret Manager access (read GITHUB_TOKEN)
resource "google_secret_manager_secret_iam_member" "dora_yaki_api_github_token" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.github_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.dora_yaki_api.email}"
}
