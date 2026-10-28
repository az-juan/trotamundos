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
cp .env.example deploy/.env   # o crear deploy/.env a partir de .env.example
go work sync                  # sincroniza go.work con los 3 módulos + shared
```

Estructura esperada:

```
trotamundos/
├── go.work
├── units/{pater,filius,spiritus}/go.mod
├── shared/go.mod
├── docs/{api,events}/
├── deploy/{docker-compose.yml,nginx/}
└── trinity/...
```

## 3. Variables de entorno

Copiar `.env.example` a `deploy/.env` y ajustar. Variables mínimas para desarrollo:

| Variable | Unidad | Descripción |
|----------|--------|-------------|
| `PATER_PG_DSN` | pater | DSN a PostgreSQL (esquema `pater`) |
| `PATER_REDIS_URL` | pater | `redis://localhost:6379` |
| `JWT_PRIVATE_KEY_FILE` | pater | Ruta a la clave privada RS256 |
| `FILIUS_PG_DSN` | filius | DSN a PostgreSQL (esquema `filius`) |
| `FILIUS_REDIS_URL` | filius | `redis://localhost:6379` |
| `NATS_URL` | filius/spiritus | `nats://localhost:4222` |
| `SPIRITUS_PG_DSN` | spiritus | DSN a PostgreSQL (esquema `spiritus`) |
| `SPIRITUS_CH_DSN` | spiritus | `clickhouse://localhost:9000` (Fase 1) |
| `GMAIL_CLIENT_ID` / `GMAIL_CLIENT_SECRET` | spiritus | OAuth2 para Gmail API |
| `IMAP_HOST` / `IMAP_USER` / `IMAP_PASS` | spiritus | Conexión IMAP genérica |

Ver `.env.example` completo.

## 4. Levantar dependencias (Docker Compose)

```bash
cd deploy
docker compose up -d postgres redis nats
# PostgreSQL managed en Fase 0; en local usamos el contenedor postgres del compose
```

Aplicar migraciones:

```bash
migrate -path migrations/filius -database "$FILIUS_PG_DSN" up
migrate -path migrations/spiritus -database "$SPIRITUS_PG_DSN" up
migrate -path migrations/pater -database "$PATER_PG_DSN" up
```

## 5. Correr las unidades

En terminales separadas (o con `go run`):

```bash
# Terminal 1 — Filius (debe arrancar primero, es el núcleo)
cd units/filius && go run ./cmd/filius   # :8081

# Terminal 2 — Spiritus
cd units/spiritus && go run ./cmd/spiritus # :8082

# Terminal 3 — Pater (depende de Filius)
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
oapi-codegen -config units/pater/internal/gateway/oapi-codegen.yaml docs/api/openapi.yaml

# JSON Schema -> tipos Go (go-jsonschema o quicktype)
# (ver shared/event para los tipos manuales sincronizados con docs/events/events.yaml)
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
- **JWT inválido**: verificar que la clave privada de Pater coincide con el JWKS publicado en `/.well-known/jwks.json`.

> Ver también: `trinity/units/01-arquitectura-global.md` (layout del monorepo) y `trinity/vista_despliegue.md` (infra).
