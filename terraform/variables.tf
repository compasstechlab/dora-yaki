variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "asia-northeast1"
}

# ----- Resource naming -----

variable "resource_name_prefix" {
  description = "Prefix used for generated resource names such as Cloud Scheduler jobs. Use lowercase letters, numbers, and hyphens."
  type        = string
  default     = "dora-yaki"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]*[a-z0-9]$", var.resource_name_prefix))
    error_message = "resource_name_prefix must start with a lowercase letter, end with a lowercase letter or number, and contain only lowercase letters, numbers, and hyphens."
  }
}

variable "service_account_id" {
  description = "Service account ID for the backend and scheduler invoker. Must be unique in the project and at most 30 characters."
  type        = string
  default     = "dora-yaki-api"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.service_account_id))
    error_message = "service_account_id must be 6-30 characters, start with a lowercase letter, end with a lowercase letter or number, and contain only lowercase letters, numbers, and hyphens."
  }
}

variable "service_account_display_name" {
  description = "Display name for the backend service account."
  type        = string
  default     = "DORA-Yaki API"
}

variable "service_account_description" {
  description = "Description for the backend service account."
  type        = string
  default     = "Service account for dora-yaki backend (Cloud Functions + Cloud Scheduler)"
}

variable "secret_replication_locations" {
  description = "Secret Manager user-managed replication locations. Leave empty to use automatic replication."
  type        = list(string)
  default     = []
}

# ----- Datastore -----

variable "create_datastore_database" {
  description = "When true, create the default Firestore database in Datastore mode. Set false if the project already has a default database."
  type        = bool
  default     = true
}

variable "datastore_location_id" {
  description = "Firestore database location ID for Datastore mode. Defaults to region when empty."
  type        = string
  default     = ""
}

# ----- Encryption -----

variable "use_kms" {
  description = "When true, provision a Cloud KMS key for token encryption. When false, the deployment relies on ENCRYPTION_KEY_BASE64 stored in Secret Manager."
  type        = bool
  default     = false
}

variable "kms_location" {
  description = "Cloud KMS key ring location (used only when use_kms = true)."
  type        = string
  default     = "asia-northeast1"
}

variable "kms_key_ring_name" {
  description = "Cloud KMS key ring name used when use_kms = true."
  type        = string
  default     = "dora-yaki"
}

variable "kms_crypto_key_name" {
  description = "Cloud KMS crypto key name used when use_kms = true."
  type        = string
  default     = "tokens"
}

# ----- Cloud Run service URL -----

variable "backend_service_url" {
  description = "Absolute base URL of the deployed backend (e.g. https://dora-yaki-api-xyz.a.run.app). Used by Cloud Scheduler to construct job URLs. Leave empty to skip scheduler job creation."
  type        = string
  default     = ""
}

variable "scheduler_auth_key" {
  description = "Shared auth key sent by Cloud Scheduler in X-DORA-YAKI-JOB-KEY. Must match the JOB_AUTH_KEY runtime secret. Leave empty to skip scheduler job creation."
  type        = string
  default     = ""
  sensitive   = true
}

# ----- Cloud Scheduler -----

variable "sync_schedule" {
  description = "Cron expression for the periodic repository sync job."
  type        = string
  default     = "*/15 * * * *"
}

variable "sync_job_name" {
  description = "Cloud Scheduler job name for repository sync. Defaults to <resource_name_prefix>-sync when empty."
  type        = string
  default     = ""
}

variable "permission_check_schedule" {
  description = "Cron expression for the daily permission-check job."
  type        = string
  default     = "0 3 * * *"
}

variable "permission_check_job_name" {
  description = "Cloud Scheduler job name for permission checks. Defaults to <resource_name_prefix>-permission-check when empty."
  type        = string
  default     = ""
}

variable "scheduler_time_zone" {
  description = "IANA time zone used by Cloud Scheduler jobs."
  type        = string
  default     = "Asia/Tokyo"
}
