locals {
  services = [
    "api-gateway",
    "auth-service",
    "cart-service",
    "product-service",
    "order-service",
    "user-service",
    "offer-service",
    "notification-service",
    "analytics-ingest-service",
    "attribution-service",
    "audit-service",
    "feature-flag-service",
    "conversion-webhook",
    "redirect-service",
    "landing-service",
    "storefront-service",
    "storefront-web",
    "admin-backoffice-service",
    "reporting-service"
  ]
}

module "service_irsa_roles" {
  for_each = toset(local.services)

  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.30"

  role_name = "${each.key}-irsa"

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = [
        "apps-core:${each.key}",
        "apps-public:${each.key}",
        "apps-async:${each.key}"
      ]
    }
  }
}
