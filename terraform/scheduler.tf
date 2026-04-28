# Cloud Scheduler jobs.
#
# Both jobs are created only when var.backend_service_url and
# var.scheduler_auth_key are set. Set the service URL after the first backend
# deploy, and set scheduler_auth_key to the same value as the JOB_AUTH_KEY
# runtime secret.

# Periodic repository sync (one repo per invocation; backend picks the
# longest-stale one).
resource "google_cloud_scheduler_job" "sync" {
  count     = local.scheduler_enabled ? 1 : 0
  project   = var.project_id
  region    = var.region
  name      = local.sync_job_name
  schedule  = var.sync_schedule
  time_zone = var.scheduler_time_zone

  http_target {
    http_method = "PUT"
    uri         = "${var.backend_service_url}/api/job/sync"
    headers = {
      "X-DORA-YAKI-JOB-KEY" = var.scheduler_auth_key
    }

    oidc_token {
      service_account_email = google_service_account.dora_yaki_api.email
      audience              = var.backend_service_url
    }
  }

  depends_on = [google_project_service.cloudscheduler]
}

# Daily permission check that refreshes RepositoryAccess for every (user, repo).
resource "google_cloud_scheduler_job" "permission_check" {
  count     = local.scheduler_enabled ? 1 : 0
  project   = var.project_id
  region    = var.region
  name      = local.permission_check_job_name
  schedule  = var.permission_check_schedule
  time_zone = var.scheduler_time_zone

  http_target {
    http_method = "PUT"
    uri         = "${var.backend_service_url}/api/job/permission-check"
    headers = {
      "X-DORA-YAKI-JOB-KEY" = var.scheduler_auth_key
    }

    oidc_token {
      service_account_email = google_service_account.dora_yaki_api.email
      audience              = var.backend_service_url
    }
  }

  depends_on = [google_project_service.cloudscheduler]
}
