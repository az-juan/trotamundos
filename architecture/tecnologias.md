| Categoría | Tecnología | Versión / Detalle | Rol en el sistema |
|-----------|-----------|-------------------|-------------------|
| **Lenguaje backend** | Go | 1.23+ | Servicios, Gateway, Web BFF, tooling |
| **Lenguaje iOS** | Swift | 5.9+ | App nativa iOS |
| **Lenguage Android** | Kotlin | 1.9+ | App nativa Android |
| **UI iOS** | SwiftUI + MVVM | — | Interfaz nativa iOS |
| **UI Android** | Jetpack Compose + MVVM + Hilt | — | Interfaz nativa Android |
| **Web templating** | templ | latest | Templates compilados a `.go` |
| **Web interactividad** | htmx | 1.9+ | Swap de fragmentos, SSE, polling |
| **Web CSS** | Tailwind CSS | 3.x | Utilidades CSS → build estático |
| **Web HTTP server** | Chi | v5 | Router del BFF web |
| **Gateway framework** | Chi + oapi-codegen | v5 | Routing, auth, rate limit, proxy |
| **Broker** | NATS JetStream | 2.10+ | Event bus entre servicios |
| **DB OLTP** | PostgreSQL | 16 | Reservas, viajes, usuarios, whitelists |
| **DB Cache** | Redis | 7 | Sesiones, rate limit, dedup, idempotencia |
| **DB OLAP** | ClickHouse | 24.x | Analítica, eventos, pre-agregados |
| **Driver PG** | pgx | v5 | Conexión pool a PostgreSQL |
| **Driver Redis** | go-redis | v9 | Cliente Redis |
| **Driver CH** | clickhouse-go | v2 | Batch insert + queries analíticas |
| **Cliente NATS** | nats.go | latest | Pub/Sub + JetStream |
| **IMAP client** | emersion/go-imap | v2 | Conexión IMAP (Outlook, genérico) |
| **Gmail API** | google.golang.org/api/gmail | v1 | Push + labels para Gmail |
| **OAuth2** | golang.org/x/oauth2 | latest | Auth con Gmail, social, APIs B2B |
| **Retry** | hashicorp/go-retryablehttp | latest | Backoff + jitter para APIs externas |
| **Circuit breaker** | sony/gobreaker | latest | Aislamiento por proveedor |
| **Rate limiter** | golang.org/x/time/rate | latest | Token bucket por proveedor |
| **MIME parsing** | net/mail | stdlib | Parseo de headers y body |
| **Tracing** | OpenTelemetry (Go SDK) | latest | Distributed tracing |
| **Métricas** | Prometheus client | latest | Exponer métricas `/metrics` |
| **CI/CD** | GitHub Actions | latest | Build, test, deploy |
| **Orquestación (Fase 0)** | Docker Compose | v2 | Containers en Droplet |
| **Reverse proxy** | Nginx | 1.25+ | TLS termination, routing |
| **TLS** | Let's Encrypt (Certbot) | — | Certificados automáticos |
| **Infra (Fase 0)** | DigitalOcean Droplets | 8 GB / 4 vCPU | Hosting |
| **Infra (Fase 2)** | K3s + Terraform + ArgoCD | — | K8s ligero + IaC + GitOps |
| **Dashboard datos** | Metabase | latest | Queries ad-hoc del equipo de datos |
| **Contratos** | OpenAPI 3.1 | — | Spec-driven: API + eventos |
| **Event schemas** | JSON Schema | — | Documentación de payloads NATS |
