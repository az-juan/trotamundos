# Trotamundos — Tecnologías (Trinidad)

> Mismo stack que la arquitectura de microservicios (`../architecture/tecnologias.md`). **No hay
> cambios de lenguaje, framework ni almacén**; la Trinidad solo **re-particiona** esas tecnologías en
> 3 unidades. La columna «Unidad / Cambio» indica dónde vive cada tecnología y qué cambia respecto de `main`.

| Categoría | Tecnología | Versión / Detalle | Rol en el sistema | Unidad / Cambio |
|-----------|-----------|-------------------|-------------------|-----------------|
| **Lenguaje backend** | Go | 1.23+ | Todo el backend | Igual. `go.work` reúne **3 módulos de unidad** + `shared` |
| **Lenguaje iOS** | Swift | 5.9+ | App nativa iOS | Igual (consume API del Gateway) |
| **Lenguaje Android** | Kotlin | 1.9+ | App nativa Android | Igual |
| **UI iOS** | SwiftUI + MVVM | — | Interfaz nativa iOS | Igual |
| **UI Android** | Jetpack Compose + MVVM + Hilt | — | Interfaz nativa Android | Igual |
| **Web templating** | templ | latest | Templates compilados a `.go` | **Pater** (módulo `web-bff`) |
| **Web interactividad** | htmx | 1.9+ | Swap de fragmentos, SSE, polling | **Pater** (módulo `web-bff`) |
| **Web CSS** | Tailwind CSS | 3.x | Utilidades CSS → build estático | **Pater** (módulo `web-bff`) |
| **Web HTTP server** | Chi | v5 | Router del Web BFF | **Pater** (módulo `web-bff`) |
| **Mobile BFF** | Go · Chi (JSON) | v5 | Agregación JSON para app nativa | **Pater** (módulo `mobile-bff`) — *antes deployable propio (ADR-013), ahora módulo dentro de Pater* |
| **Gateway framework** | Chi + oapi-codegen | v5 | Routing, auth, rate limit, proxy | **Pater** (módulo `gateway`) — *antes deployable propio (ADR-006), ahora módulo dentro de Pater* |
| **Broker** | NATS JetStream | 2.10+ | Event bus **solo en la arista Filius↔Spiritus** | **Spiritus** — *se reduce el número de subjects/consumidores vs. `main` (ya no lo usan Gateway/BFF)* |
| **DB OLTP** | PostgreSQL | 16 | Reservas, viajes, usuarios, whitelists | **Compartido** por Pater/Filius con **segregación por esquema** (ADR-104) |
| **DB Cache** | Redis | 7 | Sesiones, rate limit, dedup, idempotencia | **Compartido** por las 3 unidades (ADR-104) |
| **DB OLAP** | ClickHouse | 24.x | Analítica, eventos, pre-agregados | **Spiritus** (dueño exclusivo) |
| **Driver PG** | pgx | v5 | Pool a PostgreSQL | En Filius y Spiritus (cada unidad con su cliente) |
| **Driver Redis** | go-redis | v9 | Cliente Redis | Las 3 unidades |
| **Driver CH** | clickhouse-go | v2 | Batch insert + queries analíticas | **Spiritus** |
| **Cliente NATS** | nats.go | latest | Pub/Sub + JetStream | Filius (producer) y Spiritus (consumer/producer) — *solo 2 de 3 unidades lo usan* |
| **IMAP client** | emersion/go-imap | v2 | Conexión IMAP (Outlook, genérico) | **Spiritus** (módulo `email-ingest`) |
| **Gmail API** | google.golang.org/api/gmail | v1 | Push + labels para Gmail | **Spiritus** (módulo `email-ingest`) |
| **OAuth2** | golang.org/x/oauth2 | latest | Auth con Gmail, social, APIs B2B | **Spiritus** (correo) y **Pater** (login, futuro) |
| **Retry** | hashicorp/go-retryablehttp | latest | Backoff + jitter para APIs externas | **Filius** (módulo `integration`) |
| **Circuit breaker** | sony/gobreaker | latest | Aislamiento por proveedor | **Filius** (módulo `integration`) |
| **Rate limiter** | golang.org/x/time/rate | latest | Token bucket por proveedor | **Filius** (módulo `integration`) y **Pater** (rate limit por cliente) |
| **MIME parsing** | net/mail | stdlib | Parseo de headers y body | **Spiritus** (módulo `email-ingest`) |
| **Tracing** | OpenTelemetry (Go SDK) | latest | Distributed tracing | Las 3 unidades (correlation-id en REST y eventos) |
| **Métricas** | Prometheus client | latest | Exponer métricas `/metrics` | Las 3 unidades |
| **CI/CD** | GitHub Actions | latest | Build, test, deploy | **3 pipelines** (uno por unidad, con template compartido) — *antes 7* |
| **Orquestación (Fase 0)** | Docker Compose | v2 | **3 contenedores** + NATS en un Droplet | Igual (ADR-012) pero con 3 imágenes en vez de 7 |
| **Reverse proxy** | Nginx | 1.25+ | TLS termination, routing | Igual. Enruta a **Pater** (único expuesto) |
| **TLS** | Let's Encrypt (Certbot) | — | Certificados automáticos | Igual |
| **Infra (Fase 0)** | DigitalOcean Droplets | 8 GB / 4 vCPU | Hosting de los 3 contenedores | Igual (ADR-012) |
| **Infra (Fase 1)** | 2.º Droplet (16 GB / 8 vCPU) | — | ClickHouse + Metabase (dueño: Spiritus) | Igual |
| **Infra (Fase 2)** | K3s + Terraform + ArgoCD | — | K8s ligero + IaC + GitOps | Igual; la Trinidad migra **3 despliegues**, no 7 |
| **Dashboard datos** | Metabase | latest | Queries ad-hoc del equipo de datos | **Spiritus** |
| **Contratos** | OpenAPI 3.1 | — | Spec-driven: API del Gateway/BFF | `docs/api/openapi.yaml` (sin cambio) |
| **Event schemas** | JSON Schema | — | Payloads NATS (`docs/events/events.yaml`) | Se **reduce** el catálogo de subjects (ADR-103) |

## Cambios respecto de `main` (resumen)

1. **Gateway, Web BFF y Mobile BFF dejan de ser deployables independientes** y pasan a ser **módulos
   dentro de la unidad Pater** (ADR-102). El principio *one BFF per experience* (ADR-013) se conserva
   como **módulos** en vez de como servicios.
2. **Itinerary e Integration se unen en la unidad Filius** (ambos per-request/OLTP; ADR-102).
3. **Email Ingest y Analytics se unen en la unidad Spiritus** (ambos batch/async; ADR-102).
4. **NATS se restringe a la arista Filius↔Spiritus**; dentro de cada unidad la comunicación es en
   proceso (ADR-103).
5. **PostgreSQL se comparte con segregación por esquema** en lugar de un `go.mod`/DB por servicio
   (ADR-104). Es la desviación deliberada y justificada del principio «sin DB compartida» de `main`.
