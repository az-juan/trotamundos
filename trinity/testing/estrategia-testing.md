# Trotamundos — Estrategia de testing (Trinidad)

> Deriva de Newman ch.9 (pirámide) y de `units/01` §5.4. Mismo enfoque que `main`, adaptado a 3 unidades.

## 1. Pirámide (proporción ~10× por nivel)

```
        /\
       /E2E\        <- pocos ("training wheels")
      /------\
     /Contract\     <- decenas (Pact + OpenAPI/JSON-Schema)
    /----------\
   /   Unit     \   <- cientos (dominio puro)
  /--------------\
```

- **Unitarios (×10):** dominio puro, sin infra. La ventaja del hexagonal: `domain` y `usecase` se testean sin PG/Redis/NATS. Framework: `testing` stdlib + `testify/require` + `go-cmp` para diffs.
- **De servicio / contrato (×1):** puertos con adaptadores fake (in-memory repo, fake publisher). **Consumer-Driven Contracts (Pact)** entre Pater (Gateway/BFF) y Filius, y entre consumidores de eventos.
- **E2E (pocos):** `docker compose` con las 3 unidades + DBs efímeras. Solo flujos críticos (login → ver viaje → disrupción → notificación). En Fase 1 se añade: consulta de estado → derivación a agencia → `assistance.resolved`.
- **En producción:** smoke tests + verificación de SLO (Newman: invertir en MTTR, no en más tests pre-prod).

## 2. Qué se testea por unidad

| Unidad | Unitarios | Contrato | Integración |
|--------|-----------|----------|-------------|
| Pater | render templ/HTMX, sesión cookie, rate limit | `Store` fake con sqlc (ADR-111) | fragmentos HTMX + Hurl E2E |
| Filius | state machine Booking, use cases | eventos (JSON Schema) | PG (testcontainers), NATS |
| Spiritus | parseo de correo (regex→template), dedup | eventos (JSON Schema) | IMAP mock, ClickHouse (Fase 1) |
| Pater · assistant (Fase 1) | `IntentClassifier` (fake), redacción de PII, `AssistanceCase` state machine | OpenAPI `/assistant/*` + Pact Pater→Filius | LLM **fake** (nunca live en CI), Redis efímero |

## 3. Tests de contrato (SDD)

- **OpenAPI:** en Fase 0 es contrato de referencia para el BFF móvil (ADR-111); no genera handlers. Su validación con `schemathesis`/`prism` se reintroduce en Fase 1.
- **Eventos:** `shared/event` valida los payloads contra `fase_0/docs/events/events.yaml` en boot y en CI (`jsonschema`).
- **Pact:** se reintroduce en Fase 1 cuando vuelva la arista REST Pater↔Filius; en Fase 0 no hay cliente HTTP interno que contratar.

- El mapa se prueba en dos niveles: el handler HTML referencia `/map/image` y verifica que no se añada `<script>`; Hurl comprueba que `/map/image` devuelve `image/png` (ADR-112).
- Toda respuesta HTMX se valida con swaps OOB de `<title>` y `#site-nav`: el enlace `is-current` y el título deben cambiar con la ruta sin redibujar el documento.

## 3.1. sqlc, HTMX y NATS (ADR-111)
- No hay tests de API JSON: la capa JSON fue eliminada. Los handlers de Pater se prueban con un `Store` fake y comprueban fragmentos HTMX y la cookie firmada.
- Las consultas se generan con `sqlc generate` desde `db/queries/*.sql` y `db/schema.sql`; `shared/db` es el código generado (no editar a mano).


## 4bis. Pruebas E2E con Hurl (Fase 0 + Inicio Parcial Fase 1)

Con el compose levantado, `./scripts/hurl.sh` ejecuta **8 archivos** E2E (`fase_0/tests/hurl/*.hurl` y `fase_1/tests/hurl/*.hurl`):

| Archivo | Cobertura |
|---------|-----------|
| `smoke.hurl` | `/healthz` de los 3 deployables, 404 de superficies JSON eliminadas (ADR-111) y 200 de `/assistant` (ADR-113) |
| `pages.hurl` | Páginas SSR, CSS madera y `htmx.min.js` servido localmente |
| `htmx.hurl` | fragmentos `/trips`, `/insights` y `/demo/ingest` con `HX-Request: true` |
| `auth-preferences.hurl` | cookie firmada HttpOnly, preferencias persistidas con sqlc y logout |
| `crud.hurl` | alta de viaje/reserva, compartir, ver itinerario y archivar (FR-4/5/6/8) |
| `mapa.hurl` | página del mapa y PNG real `/map/image` (FR-12, ADR-112) |
| `integracion.hurl` | webhook sin firma → 401, con HMAC → 202, y salud final (ADR-007) |
| `fase_1/tests/hurl/assistant.hurl` | consulta Nivel 1, reprogramación guiada (FR-6) y derivación a agencia preferida (FR-11, ADR-109/110/113) |

```bash
cd fase_0/deploy && docker compose up -d --build
cd ../.. && ./scripts/hurl.sh
```

El script calcula la firma HMAC del payload y usa IDs/tokens leídos de PostgreSQL, por lo que es
idempotente: el webhook degrada y restaura la reserva `PNR-1930` en la misma corrida.

## 4. Flaky tests

- Se **cuarentenan** (no se ignoran): un test flaky que se ignora normaliza la desviación (Newman ch.9).
- Causas típicas: tiempo real, orden de goroutines, red. Mitigación: `testing/synctest` (Go 1.24+), relojes inyectables, contenedores efímeros con `testcontainers-go`.

## 5. Cobertura y métricas

- Objetivo: **≥ 80% en `domain` y `usecase`**; adaptadores se cubren por integración.
- Métrica en CI: `go test -coverprofile` + `gocov`/`gocov-html`. Bloquea el merge si baja del umbral en `domain`.

## 6. Convenciones

- Archivos: `*_test.go` junto al código (`foo_test.go`).
- Nombres: `Test<Unidad>_<Escenario>_<Resultado>` (ej. `TestFilius_BookingState_DiscoveredToConfirmed`).
- Fixtures: `testdata/` junto al paquete; golden files con `-update` flag.
- No mocks de frameworks pesados: usar fakes manuales (Newman: stubs > mocks para claridad).

## 7. Asistente Virtual (Fase 1, ADR-109/110)

- **Nunca llama al LLM real en CI:** el puerto `IntentClassifier` se inyecta con un fake; los fixtures cubren `status_query`, `reschedule_guidance`, `minor_issue` y `complex_handoff`.
- **Redacción de PII:** test unitario que verifica que el payload enviado al clasificador no contenga nombres, documentos ni correos.
- **Fallback:** simular error/tiempo de espera del adaptador LLM y comprobar `ASSISTANT_DEGRADED` + derivación disponible.
- **Contratos:** el OpenAPI de `/assistant/*` y `/internal/assistance/*` se valida con `schemathesis`/`prism`; los eventos `assistance.*` contra `events.yaml`.
- **Métricas QA-16/QA-17:** test de agregación con eventos sintéticos, comprobando que el ratio ignora textos y solo usa `tier1_resolved`.

## 8. Comandos

```bash
# Unitarios
go test ./units/... ./shared/...

# Con cobertura
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Contract tests
go test ./... -tags=contract -v

# E2E con Hurl (requiere `docker compose up -d --build`)
./scripts/hurl.sh
```
