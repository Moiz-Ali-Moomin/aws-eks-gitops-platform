# This module defines an isolated "Cell" in the architecture.
# A Cell is a fully autonomous Kubernetes cluster capped at serving ~500k users.
# Configuration drift in this cluster will NEVER affect other cells globally.

variable "project_id" { type = string }
variable "region" { type = string }
variable "cell_id" { type = string }
variable "max_pods_per_node" { default = 110 }

# Dedicated VPC Subnetwork for this specific Cell to contain IP exhaustion blast radius
resource "google_compute_subnetwork" "cell_subnet" {
  name          = "vpc-sub-cell-${var.cell_id}-${var.region}"
  ip_cidr_range = "10.${var.cell_id}.0.0/16"
  region        = var.region
  network       = "projects/${var.project_id}/global/networks/ecommerce-vpc"
  
  secondary_ip_range {
    range_name    = "pod-ranges-${var.cell_id}"
    ip_cidr_range = "10.${var.cell_id + 100}.0.0/14"
  }
  secondary_ip_range {
    range_name    = "service-ranges-${var.cell_id}"
    ip_cidr_range = "10.${var.cell_id + 200}.0.0/20"
  }
}

# The Isolated Regional GKE Cluster
resource "google_container_cluster" "cell" {
  name     = "ecommerce-cell-${var.cell_id}"
  location = var.region

  # Use Regional Autopilot to eliminate node management overhead for 20+ clusters
  enable_autopilot = true

  network    = google_compute_subnetwork.cell_subnet.network
  subnetwork = google_compute_subnetwork.cell_subnet.name

  ip_allocation_policy {
    cluster_secondary_range_name  = google_compute_subnetwork.cell_subnet.secondary_ip_range[0].range_name
    services_secondary_range_name = google_compute_subnetwork.cell_subnet.secondary_ip_range[1].range_name
  }

  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false
    master_ipv4_cidr_block  = "172.16.${var.cell_id}.0/28"
  }

  # Fleet Workload Identity allows cross-cell authentication to central DBs
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  # Anthos Fleet registration for Centralized Hub GitOps management
  fleet {
    project = var.project_id
  }
}

output "cluster_name" {
  value = google_container_cluster.cell.name
}

output "cluster_endpoint" {
  value     = google_container_cluster.cell.endpoint
  sensitive = true
}
