# Trotamundos — Guía de desarrollo local (Fase 0)

> **Rama:** `main` (consolidada) · **Objetivo:** levantar las 3 unidades + dependencias en local para desarrollar y probar.

## 1. Prerrequisitos

- Go 1.23+ (`go version`)
- Docker + Docker Compose v2 (`docker compose version`)
- Node 20+ (solo para validar mermaid / tooling, no para el backend)
- `golangci-lint` (opcional, recomendado)
- `oapi-codegen` y `sqlc`/`golang-migrate` (para regenerar código)

## 2. Clonar y preparar el workspace

```bash
git clone https://github.com/az-juan/trotamundos.git
cd trotamundos
cp fase_0/.env.example fase_0/deploy/.env   # o crear fase_0/deploy/.env a partir de fase_0/.env.example
go work sync                  # sincroniza go.work con los 3 módulos + shared
```

Estructura esperada:

```
trotamundos/
├── go.work
├── units/{pater,filius,spiritus}/go.mod
├── shared/go.mod
├── fase_0/docs/{api,events}/
├── fase_0/deploy/{docker-compose.yml,nginx/}
└── trinity/...
```

## 3. Variables de entorno

Copiar `fase_0/.env.example` a `fase_0/deploy/.env` y ajustar. Variables mínimas para desarrollo:

| Variable | Unidad | Descripción |
|----------|--------|-------------|
| `PATER_PG_DSN` | pater | DSN a PostgreSQL con rol `pater_app` (esquema `pater`, ADR-108) |
| `PATER_REDIS_URL` | pater | `redis://localhost:6379/0` |
| `SESSION_SECRET` | pater | Clave HMAC de la cookie de sesión (Fase 0; en Fase 1 se sustituye por OIDC) |
| `FILIUS_PG_DSN` | filius | DSN a PostgreSQL con rol `filius_app` (esquema `filius`, ADR-108) |
| `FILIUS_REDIS_URL` | filius | `redis://localhost:6379/1` |
| `NATS_URL` | filius/spiritus | `nats://localhost:4222` |
| `SPIRITUS_PG_DSN` | spiritus | DSN a PostgreSQL con rol `spiritus_app` (esquema `spiritus`, ADR-108) |
| `SPIRITUS_ANALYTICS_DRIVER` | spiritus | `postgres` en Fase 0 → `clickhouse` en Fase 1 (ADR-107) |
| `SPIRITUS_CH_DSN` | spiritus | `clickhouse://localhost:9000` (activo en Fase 1, ADR-107) |
| `GMAIL_CLIENT_ID` / `GMAIL_CLIENT_SECRET` | spiritus | OAuth2 para Gmail API |
| `IMAP_HOST` / `IMAP_USER` / `IMAP_PASS` | spiritus | Conexión IMAP genérica |

Ver `fase_0/.env.example` completo.

## 4. Levantar dependencias (Docker Compose)

```bash
cd deploy
docker compose up -d postgres redis nats
# En local usamos el contenedor PostgreSQL del compose; `init.sql` crea esquemas, roles, seeds y grants
```

Aplicar migraciones:

```bash
docker compose exec -T postgres psql -U trotamundos -d trotamundos -f /dev/stdin < fase_0/deploy/postgres/init.sql
```

## 5. Correr las unidades

En terminales separadas (o con `go run`):

```bash
# Terminal 1 — Filius (debe arrancar primero, es el núcleo)
cd units/filius && go run ./cmd/filius   # :8081

# Terminal 2 — Spiritus
cd units/spiritus && go run ./cmd/spiritus # :8082

# Terminal 3 — Pater (depende de PostgreSQL con `PATER_PG_DSN`, no de Filius en Fase 0)
cd units/pater && go run ./cmd/pater       # :8080
```

O con Docker Compose (las 3 unidades + deps):

```bash
cd deploy && docker compose up -d --build
```

## 6. Verificación rápida (smoke)

```bash
curl http://localhost:8080/healthz          # Pater
curl http://localhost:8081/healthz          # Filius
curl http://localhost:8082/healthz          # Spiritus
curl http://localhost:8222/healthz          # NATS monitoring
```

Login de prueba (usuario seed en `migrations/*/seed/`):

```bash
curl -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"dev@trotamundos.app","password":"dev"}'
```

## 7. Tests

```bash
# Unitarios (dominio puro, sin infra)
go test ./units/.../internal/.../domain/...

# De servicio / contrato (puertos con adaptadores fake)
go test ./units/...

# Contract tests (OpenAPI + JSON Schema) — requiere oapi-codegen + jsonschema
go test ./... -tags=contract

# E2E (docker compose con las 3 unidades + DBs efimeras)
cd deploy && docker compose -f docker-compose.e2e.yml up --abort-on-container-exit
```

## 8. Regenerar código desde contratos

```bash
# OpenAPI -> tipos Go (oapi-codegen)
oapi-codegen -config units/pater/internal/gateway/oapi-codegen.yaml fase_0/docs/api/openapi.yaml

# JSON Schema -> tipos Go (go-jsonschema o quicktype)
# (ver shared/event para los tipos manuales sincronizados con fase_0/docs/events/events.yaml)
```

## 9. Convenciones de desarrollo

- **Trunk-based**: ramas < 1 día, PRs pequeños.
- **Commits**: formato `tipo(scope): descripción` (feat, fix, docs, refactor, test, chore).
- **No compartir código de dominio** entre unidades: si un concepto se comparte, pertenece a una unidad o debe extraerse (Newman ch.5, DRY perils).
- **Toda escritura expuesta por fuera** acepta `Idempotency-Key` (Redis, TTL 24 h).
- **Correlation-ID** propagado en REST y eventos (`X-Correlation-ID`).
- **Errores**: usar el envelope `Error` de OpenAPI (`code`, `message`, `correlation_id`).

## 10. Troubleshooting

- **Filius no arranca**: verificar `FILIUS_PG_DSN` y que las migraciones se aplicaron.
- **Pater devuelve 502**: Filius no responde → verificar que Filius está up y el timeout (ADR-103).
- **NATS sin JetStream**: asegurar `command: ["-js"]` en el servicio nats del compose.
- **Sesión inválida**: verificar `SESSION_SECRET` y que la cookie `trotamundos_session` no esté vencida (ADR-111).

## 12. Despliegue completo con Docker Compose

```bash
cd fase_0/deploy
docker compose up -d --build      # construye 3 imágenes Go y levanta 7 servicios
docker compose ps                 # salud de nginx/pater/filius/spiritus/nats/redis/postgres
curl http://localhost:8088/healthz
```

- La interfaz se sirve en **`http://localhost:8088`** (Nginx → Pater); Pater también está expuesto en `:8090`.
- Los puertos host se pueden cambiar con `NGINX_HTTP_PORT`, `NGINX_HTTPS_PORT`, `PATER_HOST_PORT` y `POSTGRES_HOST_PORT`.
- `env_file` es opcional: si no existe `fase_0/deploy/.env`, la aplicación usa valores por defecto.
- Verificación funcional: abrir el panel, pulsar **Buzones** y **Simular correo**; Spiritus publica
  `booking.discovered`, Filius crea la reserva y Pater actualiza el boletín y el listado con swaps OOB.
- Sin API JSON (ADR-111): los handlers consultan PostgreSQL con **sqlc** y renderizan HTML/HTMX. Las consultas viven en `db/queries/*.sql` y el código generado en `shared/db/`.`PATER_PG_DSN` es obligatorio.
- Mapa: `/map/image` genera un PNG real con `go-staticmaps` + teselas OSM en caché desde `db/queries/trips.sql` (`ListGeoPoints`); requiere `lat`/`lon` en `filius.bookings.details`.
- Vistas disponibles: `/`, `/trips/view`, `/trips/{id}`, `/insights/view`, `/mailboxes`, `/whitelists`,
  `/preferences`, `/shared/{token}`, `/nearby` y `/login`. Autenticación por cookie firmada
  (demo: `dev@trotamundos.app` / `dev`).

## 11. Prototipo ejecutable (MVP Fase 0 — CON-4)

Pater usa PostgreSQL real mediante sqlc; Filius y Spiritus conservan adaptadores en memoria y NATS para el bus asíncrono. Las tres unidades se levantan con un solo comando.

```bash
./scripts/test.sh                # templ generate + go test ./... en shared, filius, spiritus y pater
./scripts/run.sh                 # Filius :8081, Spiritus :8082, Pater :8090
PATER_PORT=8091 ./scripts/run.sh # si :8090 está ocupada por otro proyecto local
```

- **Pater** sirve la interfaz en `http://localhost:8090`: HTML con `templ`, estilos en `units/pater/static/css/base.css`
  (CSS Grid + Flexbox) y `htmx.min.js` servido localmente desde `units/pater/static/assets/`.
- **Sin JavaScript propio**: filtros, formularios y botones usan `hx-get`/`hx-post` con `hx-target` y swaps OOB.
- Flujo de demostración: pulsar **Simular correo** → Spiritus parsea el correo y publica `booking.discovered`
  hacia Filius → Filius crea la reserva y publica `booking.confirmed` → Pater refresca el boletín y el listado.
- Las preferencias de carga son **volátiles**: al reiniciar vuelven las dos reservas semilla. La persistencia real
  (PostgreSQL con RBAC por esquema, Redis y NATS JetStream) sigue siendo el objetivo del despliegue `fase_0/deploy/`.

> Ver también: `trinity/units/01-arquitectura-global.md` (layout del monorepo) y `trinity/vista_despliegue.md` (infra).
