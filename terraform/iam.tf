# Service account, IAM, secrets, and (optional) KMS for token-at-rest encryption.

# ----- Service account -----

# Service account for the backend API (Cloud Functions / Cloud Scheduler)
resource "google_service_account" "dora_yaki_api" {
  project      = var.project_id
  account_id   = var.service_account_id
  display_name = var.service_account_display_name
  description  = var.service_account_description
}

# Datastore access
resource "google_project_iam_member" "dora_yaki_api_datastore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.dora_yaki_api.email}"
}

# ----- Secret Manager: OAuth + session + encryption secrets -----
# Set the actual values via `gcloud secrets versions add <name> --data-file=-` or Console.

locals {
  api_secret_ids = [
    "GITHUB_OAUTH_CLIENT_ID",
    "GITHUB_OAUTH_CLIENT_SECRET",
    "AUTH_JWT_SECRET",
    "ENCRYPTION_KEY_BASE64",
    "JOB_AUTH_KEY",
  ]
}

resource "google_secret_manager_secret" "api_secrets" {
  for_each = toset(local.api_secret_ids)

  project   = var.project_id
  secret_id = each.value

  replication {
    dynamic "auto" {
      for_each = length(var.secret_replication_locations) == 0 ? [true] : []
      content {}
    }

    dynamic "user_managed" {
      for_each = length(var.secret_replication_locations) == 0 ? [] : [true]
      content {
        dynamic "replicas" {
          for_each = local.secret_replication_regions
          content {
            location = replicas.value
          }
        }
      }
    }
  }

  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_iam_member" "dora_yaki_api_secret_access" {
  for_each = google_secret_manager_secret.api_secrets

  project   = var.project_id
  secret_id = each.value.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.dora_yaki_api.email}"
}

# ----- Optional: Cloud KMS for token encryption -----
# Set var.use_kms = true to provision a KMS key and grant the service account
# encrypter/decrypter access. Wire ENCRYPTION_KMS_KEY (instead of
# ENCRYPTION_KEY_BASE64) into the runtime when this is enabled.

resource "google_kms_key_ring" "dora_yaki" {
  count    = var.use_kms ? 1 : 0
  project  = var.project_id
  name     = var.kms_key_ring_name
  location = var.kms_location

  depends_on = [google_project_service.cloudkms]
}

resource "google_kms_crypto_key" "dora_yaki_tokens" {
  count    = var.use_kms ? 1 : 0
  name     = var.kms_crypto_key_name
  key_ring = google_kms_key_ring.dora_yaki[0].id
  purpose  = "ENCRYPT_DECRYPT"

  rotation_period = "7776000s" # 90 days

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_kms_crypto_key_iam_member" "dora_yaki_api_kms" {
  count         = var.use_kms ? 1 : 0
  crypto_key_id = google_kms_crypto_key.dora_yaki_tokens[0].id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${google_service_account.dora_yaki_api.email}"
}

# ----- Cloud Scheduler invoker -----
# Allow the scheduler service account to call the Cloud Run / Cloud Functions
# endpoints with an OIDC token. The job definitions live in scheduler.tf.

resource "google_project_iam_member" "dora_yaki_api_run_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.dora_yaki_api.email}"
}
