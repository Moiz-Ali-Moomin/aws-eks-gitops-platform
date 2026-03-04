# This Terraform module provisions an equivalent Async Event Pipeline DLQ using AWS SQS.
# AWS MSK (Kafka) clients can route failed messages here.

variable "dlq_name" {
  type    = string
  default = "order-events-dlq"
}

# 1. The Dead Letter Queue (SQS)
resource "aws_sqs_queue" "dlq" {
  name                      = var.dlq_name
  message_retention_seconds = 1209600 # 14 days maximum retention

  sqs_managed_sse_enabled = true # Server side encryption
}

# 2. Main Event Queue (if using SQS instead of MSK for simpler Pub/Sub workloads)
# In this architecture, we assume heavy events use MSK, but we provision an SQS topic
# for standard async governance (equivalent to Pub/Sub fan-out).
resource "aws_sqs_queue" "main_queue" {
  name                      = "order-events-queue"
  message_retention_seconds = 345600 # 4 days
  
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = 5 # Max receive attempts before DLQ
  })

  sqs_managed_sse_enabled = true
}

# 3. SNS Topic for Pub/Sub Broadcast (Fan-out pattern)
resource "aws_sns_topic" "main_topic" {
  name = "order-events"
}

# 4. Subscribe the Main Queue to the SNS Topic
resource "aws_sns_topic_subscription" "main_subscription" {
  topic_arn = aws_sns_topic.main_topic.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.main_queue.arn
}

# Grant SNS permission to write to SQS
resource "aws_sqs_queue_policy" "sns_to_sqs" {
  queue_url = aws_sqs_queue.main_queue.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "sns.amazonaws.com"
        }
        Action   = "sqs:SendMessage"
        Resource = aws_sqs_queue.main_queue.arn
        Condition = {
          ArnEquals = {
            "aws:SourceArn" = aws_sns_topic.main_topic.arn
          }
        }
      }
    ]
  })
}
