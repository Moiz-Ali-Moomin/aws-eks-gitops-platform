# This Terraform module provisions an Enterprise-grade Async Event Pipeline using GCP Pub/Sub.
# It replaces raw Kafka with a serverless, horizontally scaling event mesh that natively
# supports Dead-Letter routing and Schema enforcement to prevent poison pill consumer crashes.

variable "project_id" { type = string }
variable "topic_name" { default = "order-events" }

# 1. The Dead Letter Queue (DLQ) Topic
resource "google_pubsub_topic" "dlq" {
  name    = "${var.topic_name}-dlq"
  project = var.project_id
  
  message_storage_policy {
    allowed_persistence_regions = ["us-central1", "us-east1"]
  }
}

# 2. Main Event Topic with strict Protobuf Schema verification enabled
resource "google_pubsub_topic" "main_topic" {
  name    = var.topic_name
  project = var.project_id

  depends_on = [google_pubsub_schema.order_schema]

  schema_settings {
    schema   = "projects/${var.project_id}/schemas/OrderPlaced"
    encoding = "JSON" # Allows JSON payloads that strictly match the Protobuf definition
  }
}

# 3. Main Consumer Subscription with Automated DLQ Routing
resource "google_pubsub_subscription" "main_consumer" {
  name    = "${var.topic_name}-notification-subscriber"
  topic   = google_pubsub_topic.main_topic.name
  project = var.project_id

  # If a downstream service panics while parsing the event 5 times,
  # Pub/Sub automatically ejects the event to the DLQ to prevent Head-of-Line blocking.
  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dlq.id
    max_delivery_attempts = 5
  }

  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s" # Exponential backoff to protect downstream DB connection pools
  }

  enable_message_ordering    = true # Maintain causal ordering by `ordering_key` (tenant_id)
  enable_exactly_once_delivery = true # Required for idempotent financial transactions
}
