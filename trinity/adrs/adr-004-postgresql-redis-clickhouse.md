## ADR-004: PostgreSQL + Redis + ClickHouse

> ⚠️ **Procedencia:** este ADR describe la propuesta original de microservicios (`main`). La decisión vigente para la Trinidad es ADR-101..105. Se conserva como procedencia y adaptación; no es la decisión activa.


**Status**: Aceptado
**Fecha**: 2026-10-26
**Decidido por**: Equipo de arquitectura

> **Adaptación a Trinidad (ADR-104):** PostgreSQL se **comparte con segregación por esquema** (`pater`/`filius`/`spiritus`); cada unidad escribe solo en su esquema. **ClickHouse sigue exclusivo de Spiritus** (OLAP). Redis es caché compartido.

### Contexto

El sistema tiene tres perfiles de datos claramente distintos:
- **OLTP transaccional**: reservas, viajes, usuarios, whitelists (lectura/escritura por request, < 50 ms).
- **Cache / estado efímero**: sesiones, rate limits, deduplicación de correos, idempotencia de webhooks (sub-msegundo, TTL).
- **OLAP analítico**: tendencias, destinos, cancelaciones (consultas de 5–30 s, agregaciones sobre millones de filas).

### Decisión

| Almacén | Tecnología | Uso |
|---------|-----------|-----|
| **PostgreSQL 16** | Managed (DigitalOcean) | OLTP: `trips`, `bookings`, `users`, `whitelists`, `providers` |
| **Redis 7** | Managed (DigitalOcean) | Cache: sesiones, rate limit (token bucket), dedup SHA256, idempotencia webhooks |
| **ClickHouse 24.x** | Self-hosted (segundo Droplet en Fase 1) | OLAP: eventos de viaje, pre-agregados, dashboards del equipo de datos |

### Consecuencias

| Positivas | Negativas / Riesgos |
|-----------|-------------------|
| Separación clara: las queries analíticas (30 s) no compiten con el SLA de 800 ms | 3 sistemas a operar (mitigado: managed PG + Redis, solo ClickHouse es self-hosted) |
| ClickHouse: batch insert de 5000–10000 rows, materialized views para pre-agregados | ClickHouse no soporta UPDATE/DELETE eficiente (no se necesita: es append-only) |
| Redis: sub-msegundo para dedup y rate limit | Redis no es persistente por defecto (mitigado: AOF para sesiones críticas) |
| PostgreSQL: transacciones ACID para integridad de viaje + reservas | |

