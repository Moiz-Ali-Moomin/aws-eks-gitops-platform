# ECR Repositories for all microservices
# CI/CD pushes images to ECR_REGISTRY/ecommerce-repo/<service-name>

locals {
  services = [
    "admin-backoffice-service",
    "analytics-ingest-service",
    "api-gateway",
    "attribution-service",
    "audit-service",
    "auth-service",
    "cart-service",
    "conversion-webhook",
    "feature-flag-service",
    "landing-service",
    "notification-service",
    "offer-service",
    "order-service",
    "product-service",
    "redirect-service",
    "reporting-service",
    "storefront-service",
    "storefront-web",
    "user-service",
  ]
}

resource "aws_ecr_repository" "services" {
  for_each             = toset(local.services)
  name                 = "ecommerce-repo/${each.key}"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "AES256"
  }
}

# Lifecycle policy: Keep last 30 tagged images, expire untagged after 7 days
resource "aws_ecr_lifecycle_policy" "services" {
  for_each   = aws_ecr_repository.services
  repository = each.value.name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Expire untagged images after 7 days"
        selection = {
          tagStatus   = "untagged"
          countType   = "sinceImagePushed"
          countUnit   = "days"
          countNumber = 7
        }
        action = { type = "expire" }
      },
      {
        rulePriority = 2
        description  = "Keep last 30 tagged images"
        selection = {
          tagStatus   = "tagged"
          tagPrefixList = [""]
          countType   = "imageCountMoreThan"
          countNumber = 30
        }
        action = { type = "expire" }
      }
    ]
  })
}
