output "service_account_email" {
  description = "Backend service account email."
  value       = google_service_account.dora_yaki_api.email
}

output "secret_ids" {
  description = "Secret Manager secret IDs created for backend runtime secrets."
  value       = local.api_secret_ids
}

output "kms_crypto_key_id" {
  description = "KMS crypto key resource ID when use_kms is true."
  value       = var.use_kms ? google_kms_crypto_key.dora_yaki_tokens[0].id : null
}

output "scheduler_job_names" {
  description = "Cloud Scheduler job names."
  value = {
    sync             = local.sync_job_name
    permission_check = local.permission_check_job_name
  }
}
