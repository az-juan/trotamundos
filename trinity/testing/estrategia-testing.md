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
- **E2E (pocos):** `docker compose` con las 3 unidades + DBs efímeras. Solo flujos críticos (login → ver viaje → disrupción → notificación).
- **En producción:** smoke tests + verificación de SLO (Newman: invertir en MTTR, no en más tests pre-prod).

## 2. Qué se testea por unidad

| Unidad | Unitarios | Contrato | Integración |
|--------|-----------|----------|-------------|
| Pater | agregación BFF, rate limit, render | OpenAPI (gateway) | REST a Filius (fake) |
| Filius | state machine Booking, use cases | eventos (JSON Schema) | PG (testcontainers), NATS |
| Spiritus | parseo de correo (regex→template), dedup | eventos (JSON Schema) | IMAP mock, ClickHouse (Fase 1) |

## 3. Tests de contrato (SDD)

- **OpenAPI:** `oapi-codegen` genera los handlers; el test verifica que la implementación cumple el spec (round-trip request/response). Herramienta: `schemathesis` o `prism` + `go test`.
- **Eventos:** `shared/event` valida los payloads contra `docs/events/events.yaml` en boot y en CI (`jsonschema`).
- **Pact:** Pater (consumer) genera un pact contra Filius (provider). El provider verifica el pact en su pipeline.

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

## 7. Comandos

```bash
# Unitarios
go test ./units/... ./shared/...

# Con cobertura
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Contract tests
go test ./... -tags=contract -v

# E2E
cd deploy && docker compose -f docker-compose.e2e.yml up --abort-on-container-exit
```
