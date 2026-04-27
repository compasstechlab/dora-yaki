locals {
  datastore_location_id      = var.datastore_location_id == "" ? var.region : var.datastore_location_id
  sync_job_name              = var.sync_job_name == "" ? "${var.resource_name_prefix}-sync" : var.sync_job_name
  permission_check_job_name  = var.permission_check_job_name == "" ? "${var.resource_name_prefix}-permission-check" : var.permission_check_job_name
  secret_replication_regions = toset(var.secret_replication_locations)
  scheduler_enabled          = var.backend_service_url != "" && var.scheduler_auth_key != ""
}
