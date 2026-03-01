resource "google_spanner_instance" "ecommerce_orders" {
  name         = "ecommerce-orders-db"
  config       = "nam-eur-asia1" # Multi-region spanning NA, Europe, Asia
  display_name = "Enterprise Orders Multi-Region"
  num_nodes    = 3
  
  labels = {
    "environment" = "production"
    "criticality" = "high"
  }
}

resource "google_spanner_database" "orders_db" {
  instance = google_spanner_instance.ecommerce_orders.name
  name     = "orders"
  version_retention_period = "3d" # Point-in-time recovery
  
  # Prevent accidental destruction of production data
  lifecycle {
    prevent_destroy = true
  }
}

# Grant Workload Identity Service Account access to Spanner
resource "google_project_iam_member" "spanner_access" {
  project = var.project_id
  role    = "roles/spanner.databaseUser"
  member  = "serviceAccount:order-service-sa@${var.project_id}.iam.gserviceaccount.com"
}
