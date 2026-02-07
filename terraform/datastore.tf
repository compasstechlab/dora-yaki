# Datastore composite index definitions
# Managed via google_firestore_index (Firestore in Datastore Mode)

# PullRequest: filter by repository_id + sort by created_at DESC
resource "google_firestore_index" "pull_request_repo_created" {
  project    = var.project_id
  database   = "(default)"
  collection = "PullRequest"
  query_scope = "COLLECTION_GROUP"
  api_scope   = "DATASTORE_MODE_API"

  fields {
    field_path = "repository_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "created_at"
    order      = "DESCENDING"
  }
}

# Review: filter by repository_id + sort by submitted_at DESC
resource "google_firestore_index" "review_repo_submitted" {
  project    = var.project_id
  database   = "(default)"
  collection = "Review"
  query_scope = "COLLECTION_GROUP"
  api_scope   = "DATASTORE_MODE_API"

  fields {
    field_path = "repository_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "submitted_at"
    order      = "DESCENDING"
  }
}

# Deployment: filter by repository_id + sort by created_at DESC
resource "google_firestore_index" "deployment_repo_created" {
  project    = var.project_id
  database   = "(default)"
  collection = "Deployment"
  query_scope = "COLLECTION_GROUP"
  api_scope   = "DATASTORE_MODE_API"

  fields {
    field_path = "repository_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "created_at"
    order      = "DESCENDING"
  }
}

# DailyMetrics: filter by repository_id + sort by date ASC
resource "google_firestore_index" "daily_metrics_repo_date" {
  project    = var.project_id
  database   = "(default)"
  collection = "DailyMetrics"
  query_scope = "COLLECTION_GROUP"
  api_scope   = "DATASTORE_MODE_API"

  fields {
    field_path = "repository_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "date"
    order      = "ASCENDING"
  }
}

# Sprint: filter by repository_id + sort by start_date DESC
resource "google_firestore_index" "sprint_repo_start_date" {
  project    = var.project_id
  database   = "(default)"
  collection = "Sprint"
  query_scope = "COLLECTION_GROUP"
  api_scope   = "DATASTORE_MODE_API"

  fields {
    field_path = "repository_id"
    order      = "ASCENDING"
  }

  fields {
    field_path = "start_date"
    order      = "DESCENDING"
  }
}
