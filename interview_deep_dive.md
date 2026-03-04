# Principal Staff Cloud/DevOps Architect — Interview Deep Dive

> *"Walk us through this system as if you designed it, deployed it, and are on-call for it."*

---

## 1. Project Overview — The Elevator Pitch

This is a **cloud-native e-commerce platform** built as a polyglot microservices system running on **AWS EKS**. It consists of 19 Go microservices and 1 Next.js frontend, orchestrated via ArgoCD GitOps, secured with Istio service mesh, and backed by Kafka for event streaming, PostgreSQL for persistence, and Redis for caching/sessions.

The system is designed for:
- **~100k monthly active users**
- **99.9% uptime SLA**
- **Sub-second API response times** at p99
- **Zero-downtime deployments** via Argo Rollouts canary strategy
- **AZ-resilient** within a single AWS region

---

## 2. Functional Requirements

| Requirement | Service(s) | Description |
|-------------|-----------|-------------|
| **User Registration & Auth** | `auth-service`, `user-service` | JWT-based authentication, session management via Redis, password hashing |
| **Product Catalog** | `product-service` | CRUD for products, search, filtering. Postgres-backed |
| **Shopping Cart** | `cart-service` | Redis-backed ephemeral cart. Survives session closure |
| **Order Processing** | `order-service` | Order creation, payment flow orchestration, state machine. Revenue-critical path |
| **Storefront** | `storefront-service`, `storefront-web` | Backend-for-frontend (BFF) pattern. `storefront-web` is Next.js SSR |
| **Notifications** | `notification-service` | Async event-driven notifications (email, push) triggered by Kafka events |
| **Offers & Promotions** | `offer-service` | Promotion engine, coupon validation, discount logic |
| **Landing Pages** | `landing-service` | Dynamic marketing landing pages, A/B variant serving |
| **URL Redirects** | `redirect-service` | Affiliate tracking redirects, UTM parameter processing |
| **Attribution** | `attribution-service` | Marketing attribution (click → conversion tracking) via Kafka events |
| **Analytics Ingestion** | `analytics-ingest-service` | Real-time event ingestion pipeline. Kafka consumer → data warehouse |
| **Conversion Webhooks** | `conversion-webhook` | 3rd-party conversion tracking (ClickBank postbacks) |
| **Feature Flags** | `feature-flag-service` | Runtime feature toggling without redeploy |
| **Audit Trail** | `audit-service` | Compliance logging of all state-changing operations |
| **Reporting** | `reporting-service` | Async report generation from Kafka event stream |
| **Admin Backoffice** | `admin-backoffice-service` | Internal admin tool with Metabase integration |
| **API Gateway** | `api-gateway` | Central ingress, request routing, auth token validation, rate aggregation |

---

## 3. Non-Functional Requirements

| Category | Requirement | How Achieved |
|----------|-------------|-------------|
| **Availability** | 99.9% uptime (8.7h/year downtime budget) | Multi-AZ deployment, PDB, HPA, replica≥2 for critical services |
| **Scalability** | Handle 5x traffic spikes | HPA auto-scales pods, Karpenter auto-provisions nodes |
| **Latency** | p99 < 2 seconds for user-facing APIs | Go (compiled, sub-ms overhead), Redis caching, connection pooling |
| **Security** | Fintech-grade | mTLS via Istio, IRSA for AWS credentials, ExternalSecrets for secret management, Kyverno for image verification, KMS encryption at rest |
| **Observability** | SRE golden signals | OTel traces → Tempo, metrics → Prometheus/Grafana, logs → Loki, alerts → Alertmanager |
| **Deployability** | Zero-downtime releases | Argo Rollouts canary (5% → 20% → 50% → 100%), auto-rollback on error spike |
| **Disaster Recovery** | RPO < 1 hour, RTO < 5 minutes | Postgres WAL archiving to S3, daily pg_dump, Kafka RF=3 |
| **Cost Efficiency** | Startup-budget (~$500/mo) | Graviton ARM instances, right-sized resource limits, Karpenter consolidation |

---

## 4. Architecture Philosophy & Key Tradeoffs

### Tradeoff 1: Monorepo vs Polyrepo
**Decision: Monorepo** using [go.work](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/go.work) (Go workspaces).

```
go.work          → workspace definition
services/
  shared-lib/    → shared utilities, Kafka client, logging
  auth-service/
  order-service/
  ...
```

*Why:* Single repo means atomic commits across services, shared CI pipeline, consistent dependency versions. At 19 services, polyrepo would create 19 separate CI pipelines and version matrix hell.

*Tradeoff:* CI runs all tests on every push (mitigated by monorepo optimization — only build changed services).

### Tradeoff 2: EKS vs ECS Fargate
**Decision: EKS.**

*Why:* Kafka (Strimzi) needs real nodes. Istio sidecars need Kubernetes. 19 services at ~256Mi each is cheaper on shared nodes ($55/node) than Fargate ($29/service/mo × 19 = $551).

*Tradeoff:* EKS has $73/mo fixed control plane cost. Accepted.

### Tradeoff 3: Self-Hosted Kafka vs SQS/SNS
**Decision: Kafka (Strimzi).**

*Why:* 13 services publish/consume events. Kafka provides: event replay, consumer groups, exactly-once semantics, ordered partitions. SQS is simpler but loses replay (the ability to re-process historical events).

*Tradeoff:* Kafka costs ~3 nodes worth of RAM. At higher scale this is justified by the feature set. At startup scale, SQS is a viable cost optimization.

### Tradeoff 4: Self-Hosted Postgres vs RDS
**Decision: Self-hosted (Bitnami Helm chart).**

*Why:* Runs on existing EKS nodes at $0 incremental cost. RDS adds $50-100/mo for something we already have.

*Tradeoff:* No automatic failover (requires Patroni). No point-in-time recovery (mitigated by WAL archiving to S3). Accepted at startup scale.

### Tradeoff 5: Istio vs No Mesh
**Decision: Istio, slimmed down.**

*Why:* mTLS (zero-trust networking), traffic routing for canary releases (Argo Rollouts uses Istio VirtualServices), and AuthorizationPolicies for namespace-level access control.

*Tradeoff:* Istio adds ~1.5 GB RAM in sidecar overhead. Justified by security posture and canary routing. Reduced to 1 istiod replica and minimal sidecar resources.

---

## 5. Project Structure — Every Directory Explained

### Root Files

| File | Purpose |
|------|---------|
| [Dockerfile.services](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/Dockerfile.services) | Multi-stage Go builder. Stage 1: `golang:1.24.0-bookworm` compiles any service via `--build-arg SERVICE_NAME`. Stage 2: `distroless/base-debian12:nonroot` for minimal attack surface. **Key design:** one Dockerfile for all 19 Go services |
| [go.work](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/go.work) | Go workspace file linking all 19 service modules + `shared-lib`. Enables cross-module imports without `replace` directives |
| [docker-compose.yml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/docker-compose.yml) | Local development only. Spins up Postgres, Redis, Kafka locally for developer iteration |
| [Makefile](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/Makefile) | Developer UX shortcuts: `make build`, `make test`, `make lint` |
| [sonar-project.properties](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/sonar-project.properties) | SonarQube static analysis config. Exclusions for generated code, coverage paths |

---

### `.github/workflows/` — CI/CD Pipeline

#### [ci.yml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/.github/workflows/ci.yml) — The Main Pipeline (286 lines)

```
Push to main
    │
    ▼
[1] Quality Check ──── Go test + revive linter
    │
    ├──► [2] Build Check ──── `go build ./...` (compilation gate)
    │         │
    │         ▼
    │    [3] Trivy Scan ──── CRITICAL+HIGH CVE scan, exit-code 1
    │         │
    │         ▼
    │    [4] Terraform ──── Plan + Apply (only if infra/ changed)
    │         │
    │         ▼
    │    [5] Build & Push ──── Docker build → ECR push → Cosign sign
    │         │                → Update Helm values.yaml tag → git push
    │
    ├──► [2b] SonarQube ──── Static analysis (parallel with build)
```

**Key design decisions:**
- **OIDC authentication** — no long-lived AWS keys. GitHub Actions assumes `github-actions-ecr-role` via web identity federation
- **Monorepo optimization** — only builds services whose `services/<name>/` directory changed
- **Cosign image signing** — every image pushed to ECR is cryptographically signed
- **Tag management** — CI updates [values.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/spark-values.yaml) with `tag: "<sha7>"` and pushes `[skip ci]` commit. ArgoCD detects change → deploys

#### [build-push-ecr.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/.github/workflows/build-push-ecr.yaml) — Manual Build
`workflow_dispatch` only. Bypass for emergency hotfix pushes. Same build logic, no quality gates.

---

### `argocd/` — GitOps Delivery Layer

```
argocd/
├── root/              → Root Application (app-of-apps bootstrap)
├── bootstrap/         → Initial cluster setup
├── projects/          → ArgoCD Project definitions (RBAC boundaries)
│   └── applications.yaml → Defines: platform, tooling, services projects
├── apps/              → Individual Application definitions
│   ├── services.yaml      → 19 service applications (apps-core, apps-public, apps-async)
│   ├── data.yaml          → Postgres, Redis, Kafka
│   ├── tooling.yaml       → Airflow, SonarQube, Spark, Metabase
│   ├── observability.yaml → OTel Collector
│   ├── platform.yaml      → Istio, Kyverno
│   ├── secrets.yaml       → ExternalSecret definitions
│   └── external-secrets.yaml → ESO operator + ClusterSecretStore
```

**App-of-Apps Pattern:**
```
Root Application
    │
    ├──► platform-secrets (sync-wave: 6)
    ├──► platform-external-secrets (sync-wave: 5)
    ├──► platform-otel-collector (sync-wave: 10)
    ├──► services (19 apps, sync-wave: 20)
    ├──► data (Postgres, Redis, Kafka)
    └──► tooling (Airflow, Metabase, Spark)
```

*Why sync-waves:* Secrets must exist before services start. ExternalSecrets operator must exist before ExternalSecrets are created. Ordering prevents race conditions.

**Namespace Strategy:**

| Namespace | Purpose | Services |
|-----------|---------|----------|
| `apps-core` | Revenue-critical business logic | auth, order, user, product, cart, offer, feature-flag, reporting, admin-backoffice |
| `apps-public` | Internet-facing entry points | api-gateway, storefront-service, storefront-web, landing, redirect, conversion-webhook |
| `apps-async` | Event-driven background processing | notification, analytics-ingest, attribution, audit |
| `data-postgres` | PostgreSQL stateful set | Primary + replicas |
| `data-redis` | Redis stateful set | Master + replicas |
| `kafka` | Strimzi Kafka cluster | 3 brokers + ZooKeeper |
| `tooling-airflow` | DAG orchestration | Airflow scheduler + webserver |
| `tooling-metabase` | Business intelligence | Metabase dashboards |
| `platform-observability` | Monitoring stack | Prometheus, Grafana, Loki, Tempo, Alertmanager, OTel |
| `platform-argocd` | GitOps controller | ArgoCD server + repo-server |
| `external-secrets` | Secret operator | ESO controller + ClusterSecretStore |
| `istio-system` | Service mesh control plane | istiod, IngressGateway |

*Why this separation:* Blast radius isolation. A misconfigured HPA in `apps-async` can't starve `apps-core` pods. ResourceQuotas can be applied per namespace. IRSA binds service accounts to specific namespaces.

---

### `infra/terraform/aws/` — Infrastructure as Code

| File | Purpose | Key Details |
|------|---------|------------|
| [providers.tf](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/terraform/aws/providers.tf) | AWS provider, S3 backend, DynamoDB state lock | State at `s3://ecommerce-platform-terraform-state/prod/terraform.tfstate`, encrypted, DynamoDB lock table for safe concurrent applies |
| [vpc.tf](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/terraform/aws/vpc.tf) | VPC with 3-AZ private/public subnets | `10.0.0.0/16` CIDR, NAT gateway per AZ, subnets tagged for ELB discovery (`kubernetes.io/role/elb`) |
| [eks.tf](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/terraform/aws/eks.tf) | EKS cluster v1.30 | IRSA enabled, EBS CSI driver addon (for gp3), system nodes (t4g.medium ON-DEMAND) + workload nodes (m6g.large ON-DEMAND) + Karpenter burst (SPOT) |
| [iam.tf](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/terraform/aws/iam.tf) | IRSA roles, GitHub Actions OIDC, EBS CSI IRSA | Shared `ecommerce-irsa-role` with scoped `namespace:serviceaccount` pairs. GitHub OIDC with repo-scoped subject claim |
| [sqs.tf](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/terraform/aws/sqs.tf) | SQS DLQ + SNS fan-out | Dead letter queue for failed Kafka processing, SNS topic for pub/sub broadcast pattern |

**IRSA Architecture (critical interview talking point):**

```
                    ┌──────────────────────┐
                    │  AWS Secrets Manager  │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │  ecommerce-irsa-role  │
                    │  Policies:            │
                    │  - SecretsManager RW  │
                    │  - S3 Full Access     │
                    │  - SQS Full Access    │
                    └──────────┬───────────┘
                               │
              Trust Policy: Only these SAs can assume
                               │
    ┌──────────────────────────┼──────────────────────────┐
    │                          │                          │
apps-core:auth-service   apps-public:api-gateway   external-secrets:external-secrets
apps-core:order-service  apps-public:conversion-webhook  ...15 more
```

*Why shared role:* At startup scale, per-service IAM roles create diminishing returns. All services need SecretsManager access. One role with scoped trust policy is cleaner than 19 roles.

*Interview callout:* "In a larger org, I'd create per-service roles with least-privilege policies. At this scale, the shared role with per-SA trust scoping gives 80% of the security benefit at 5% of the complexity."

---

### `infra/databases/` — Data Layer

#### [postgres/values.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/databases/postgres/values.yaml)
- Bitnami PostgreSQL Helm chart
- **Architecture:** Replication (primary + read replicas)
- **Auth:** `existingSecret: postgres-creds` → no hardcoded passwords
- **HA:** `podAntiAffinityPreset: hard` → primary and replicas never on same node
- **Tuning:** `max_connections=300`, `shared_buffers=512MB`, WAL logging, connection logging
- **Storage:** `gp3` StorageClass, 20Gi per replica
- **Backup:** CronJob at 2 AM daily + S3 upload

#### [redis/values.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/databases/redis/values.yaml)
- Bitnami Redis Helm chart
- **Architecture:** Master + replica
- **Auth:** `existingSecret: redis-secret`, key `REDIS_PASSWORD`
- **Config:** `appendonly yes`, `appendfsync everysec` (AOF persistence), `maxmemory-policy allkeys-lru`
- **Storage:** `gp3`, 5Gi
- **NetworkPolicy:** `enabled: true, allowExternal: false` → only in-cluster access

#### [migrations/atlas-job.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/databases/migrations/atlas-job.yaml)
- Atlas schema-as-code migration
- Runs as ArgoCD PreSync hook → migrates DB before new service version deploys
- **Namespace:** `data-postgres`
- **Connection:** `postgres-primary.data-postgres.svc.cluster.local:5432/orders`

---

### `infra/kafka/` — Event Streaming

- **Strimzi Kafka operator** with 3-broker cluster
- [kafka-storage.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/kafka/storage/kafka-storage.yaml) — gp3 PVCs for broker data
- `schemas/OrderPlaced.proto` — Protocol Buffer schema for type-safe event contracts
- **Topic config:** RF=3, `min.insync.replicas=2`, rack-aware (topology spread across AZs)

*Interview callout:* "We chose Protobuf over Avro because our team is Go-native. Protobuf has first-class Go code generation and stricter backward compatibility guarantees."

---

### `infra/istio/` — Service Mesh

```
istio/
├── control-plane/
│   └── istiod-ha.yaml           → Istiod deployment (2 replicas, anti-affinity)
├── ingress/
│   └── virtual-services/
│       └── ecommerce-all.yaml   → Master VirtualService routing all paths
├── resilience/
│   ├── order-service-vs.yaml    → Order service retry policy (3 attempts, 2s timeout)
│   └── order-service-dr.yaml    → DestinationRule: connection pooling, outlier detection
└── security/
    ├── peer-authentication.yaml → Mesh-wide STRICT mTLS
    ├── virtual-services-patch.yaml → Resilience patches (timeouts, retries)
    └── authorization-policies/
        └── ecommerce-authz.yaml → Allow ingress + inter-namespace traffic only
```

**mTLS flow:**
```
Client → ALB (TLS) → Istio IngressGateway → Envoy Sidecar (mTLS) → Service Pod
                                                    ↕ mTLS
                                              Other Service Pod
```

*Every pod-to-pod call is encrypted.* Not because VPC traffic is insecure, but because zero-trust means no implicit trust based on network position.

---

### `infra/secrets/` & `infra/app-secrets/` — Secret Management

```
Secrets Architecture:
                    ┌─────────────────────┐
                    │ AWS Secrets Manager  │
                    │ (source of truth)    │
                    └─────────┬───────────┘
                              │ IRSA auth
                    ┌─────────▼───────────┐
                    │ ExternalSecrets      │
                    │ Operator             │
                    │ (ClusterSecretStore) │
                    └─────────┬───────────┘
                              │ creates K8s Secrets
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
   apps-core:            data-postgres:        apps-public:
   auth-service-secrets  postgres-creds        api-gateway-secrets
   auth-jwt-secret       
   auth-redis-secret     
```

**[infra/secrets/external-secrets.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/secrets/external-secrets.yaml)** — Infra-level secrets: Redis, Kafka, Airflow, Metabase credentials
**`infra/app-secrets/`** — Per-service secrets:
- [auth-service.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/app-secrets/auth-service.yaml) → 3 ExternalSecrets (DB creds, Redis password, JWT secret)
- [api-gateway.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/app-secrets/api-gateway.yaml) → API gateway DB credentials
- [conversion-webhook.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/app-secrets/conversion-webhook.yaml) → ClickBank webhook secret
- [postgres.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/app-secrets/postgres.yaml) → Postgres admin credentials

*Interview callout:* "We never have plaintext secrets in the repo. The only sensitive data in git is the *key name* in AWS Secrets Manager — never the value. ExternalSecrets auto-refreshes every hour, so secret rotation doesn't require a redeploy."

---

### `infra/observability/` — Monitoring Stack

| Component | Purpose | Data Flow |
|-----------|---------|-----------|
| **Prometheus** | Metrics collection | Scrapes `/metrics` from all pods every 30s |
| **Grafana** | Dashboards & visualization | Queries Prometheus, Loki, Tempo |
| **Alertmanager** | Alert routing | Fires to notification-service webhook + email |
| **Loki** | Log aggregation | Fluent Bit DaemonSet → Loki |
| **Tempo** | Distributed tracing | OTel SDK → OTel Collector → Tempo |
| **OTel Collector** | Telemetry pipeline | Receives traces from services, exports to Tempo |

**All services emit:** `OTEL_EXPORTER_OTLP_ENDPOINT: otel-collector.platform-observability.svc.cluster.local:4317`

---

### `infra/scaling/` — Autoscaling

[hpa-pdb-template.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/scaling/hpa-pdb-template.yaml) — Template for HPA + PDB. Applied per-service for the 8 multi-replica services.

### `infra/rollouts/` — Progressive Delivery

[order-service-rollout.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/infra/rollouts/order-service-rollout.yaml) — Argo Rollouts canary definition:
```
5% traffic → pause 5 min → 20% → pause 10 min → 50% → pause 5 min → 100%
```
Uses Istio VirtualService + DestinationRule for traffic splitting. If error rate exceeds SLO during any pause, automatic rollback.

---

### [services/](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/Dockerfile.services) — The 19 Microservices + shared-lib

Each service follows the same structure:

```
services/<name>/
├── cmd/
│   └── main.go          → Entry point
├── internal/            → Business logic (not exported)
│   ├── handlers/        → HTTP handlers
│   ├── models/          → Domain models
│   ├── repository/      → DB access layer
│   └── service/         → Business logic layer
├── helm/
│   └── values.yaml      → Kubernetes deployment config
├── go.mod
└── go.sum
```

#### `shared-lib/` — Cross-Cutting Concerns
Shared utilities imported by all services:
- Kafka producer/consumer wrappers
- Logging middleware (structured JSON)
- OTel tracing initialization
- Health check handlers (`/healthz`, `/readyz`)
- Error response formatting

*Why a shared-lib:* Go workspaces let us import `shared-lib` without `replace` directives. Avoids duplicating 500 lines of Kafka/logging code across 19 services.

#### `services/<name>/helm/values.yaml` — Anatomy

```yaml
replicaCount: 2                    # Pod count
global:
  awsAccountId: ""                 # Injected by ArgoCD
  awsRegion: "us-east-1"
  image:
    registry: "{{ ECR }}"          # Templated ECR URL
image:
  name: auth-service               # ECR image name
  tag: "latest"                    # Overwritten by CI (SHA)
service:
  type: ClusterIP                  # Internal only
  port: 8080
env:
  DB_HOST: "postgres-primary.data-postgres.svc.cluster.local"
  REDIS_HOST: "redis-master.data-redis.svc.cluster.local"
  KAFKA_BROKERS: "ecommerce-cluster-kafka-bootstrap.kafka.svc.cluster.local:9092"
  OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector.platform-observability.svc.cluster.local:4317"
resources:
  requests: { cpu: 100m, memory: 128Mi }
  limits:   { cpu: 500m, memory: 256Mi }
serviceAccount:
  create: true
  annotations:
    eks.amazonaws.com/role-arn: "arn:aws:iam::{{ acct }}:role/ecommerce-irsa-role"
```

*All DNS names are fully qualified* — `<service>.<namespace>.svc.cluster.local` — ensuring cross-namespace resolution works correctly.

---

## 6. Data Flow — End to End

```
User → ALB → Istio GW → api-gateway → auth-service (JWT validate)
                              │
                              ├──► product-service → Postgres (read)
                              ├──► cart-service → Redis (read/write)
                              └──► order-service → Postgres (write)
                                       │
                                       ├──► Kafka: topic "order-placed"
                                       │
                    ┌──────────────────►├──► notification-service (email)
                    │                  ├──► analytics-ingest-service (data warehouse)
                    │                  ├──► attribution-service (conversion tracking)
                    │                  └──► audit-service (compliance log)
                    │
                    └── Kafka Consumer Groups (independent consumers)
```

---

## 7. What To Say In The Interview

### Opening Statement
> "I designed and built a production-grade e-commerce platform on AWS EKS — 19 Go microservices, Kafka event streaming, Istio service mesh, ArgoCD GitOps, with comprehensive observability. The architecture handles AZ failures, auto-scales under load, deploys via canary rollouts with auto-rollback, and costs under $500/month."

### When Asked "What are you most proud of?"
> "The CI/CD pipeline. Every commit triggers: unit tests, lint, security scan, SonarQube, Terraform plan, Docker build, Cosign signing, ECR push, Helm tag update, and ArgoCD auto-sync — fully automated, zero human intervention, with supply chain integrity via image signatures."

### When Asked "What would you do differently?"
> "Two things. First, I'd use the Zalando Postgres Operator instead of Bitnami for automatic failover — Patroni handles leader election and promotion automatically. Second, at this scale, I might drop Kafka for SQS/SNS initially and add Kafka back when event replay becomes a hard requirement. Kafka's operational overhead at 2-3 brokers is disproportionate to the value at <100k MAU."

### When Asked "How do you handle a region failure?"
> "We don't. This is a single-region design by constraint. The architecture is AZ-resilient — pods are spread across 3 AZs via topology constraints, NAT gateways exist per AZ, and Kafka brokers are rack-aware. For multi-region, I'd need Route 53 active-active, CockroachDB or Aurora Global, and cross-region Kafka MirrorMaker. That's a 3-5x cost multiplier that isn't justified at this scale."

### When Asked "Walk me through a deployment"
> "Developer pushes to main. CI runs 5 quality gates. Docker builds only changed services (monorepo optimization). Images are signed with Cosign and pushed to ECR. CI updates each changed service's [helm/values.yaml](file:///C:/Users/Haxor/Pictures/microservice-ecommerce-aws-cloud-eks/services/api-gateway/helm/values.yaml) with the new SHA tag and pushes a `[skip ci]` commit. ArgoCD detects the git diff, syncs the Helm release. For order-service, Argo Rollouts does a canary: 5% traffic to new version, pauses 5 minutes while Prometheus monitors error rate. If SLO holds, progresses to 20%, 50%, 100%. If error rate spikes, automatic rollback."

### When Asked "What breaks under chaos testing?"
> "Pod kills are transparent — PDB + replica≥2 ensure continuity. Node drain takes 60-90 seconds (Karpenter provisions replacement). AZ failure works because of topology spread. The weakness is Postgres — Bitnami doesn't do automatic failover. A primary crash requires manual promotion of a read replica. I'd address this with Patroni or migrating to RDS Multi-AZ."

---

## 8. Numbers To Memorize For The Interview

| Metric | Value |
|--------|-------|
| Services | 19 Go + 1 Next.js |
| Total pods (steady state) | ~31 |
| Namespaces | 12 |
| Kafka brokers | 3 |
| Postgres replicas | 1 primary + 2 read |
| Redis replicas | 1 master + 1 replica |
| CI pipeline stages | 5 (test → build → scan → terraform → push+sign) |
| Deployment strategy | Canary (5% → 20% → 50% → 100%) |
| Recovery from pod kill | 0 seconds |
| Recovery from node loss | <90 seconds |
| Recovery from AZ failure | <30 seconds |
| Monthly cost | ~$500 |
| SLA target | 99.9% |
| Image size | ~20 MB (distroless) |
