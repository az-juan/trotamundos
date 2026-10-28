# Trotamundos — Operaciones (Fase 0)

> Deriva de ADR-105 (despliegue), ADR-104 (datos) y Newman ch.10 (observabilidad) / ch.11 (seguridad).

## 1. Observabilidad (concreta)

### Logs
- Formato **JSON estructurado** con campos: `ts`, `level`, `unit`, `correlation_id`, `trace_id`, `span_id`, `msg`.
- Niveles: `debug` (dev), `info` (prod), `warn`, `error`.
- Agregación: en Fase 0, archivos rotados en el Droplet + `tail`; Fase 1: Loki/Grafana.

### Métricas Prometheus (endpoint `/metrics` en cada unidad)

| Métrica | Tipo | Unidad | Descripción |
|---------|------|--------|-------------|
| `http_requests_total` | counter | todas | por método, ruta, status |
| `http_request_duration_seconds` | histogram | todas | latencia p50/p95/p99 |
| `breaker_state` | gauge | filius | 0=closed, 1=open, 2=half-open (por proveedor) |
| `retry_attempts_total` | counter | filius | por proveedor |
| `rate_limit_hits_total` | counter | pater/filius | por cliente/proveedor |
| `nats_consumer_lag` | gauge | spiritus | lag del consumer de analytics |
| `email_parse_total` | counter | spiritus | por resultado (matched/no_match/failed) |
| `email_parse_duration_seconds` | histogram | spiritus | duración del parseo |
| `idempotency_hits_total` | counter | todas | reentregas deduplicadas |

### Tracing
- OpenTelemetry Go SDK → OTLP (Fase 1: Grafana Tempo/Jaeger).
- Cada request y evento arrastra `trace_id`/`span_id`; el `correlation_id` se propaga como atributo.

### SLO de referencia
- Disponibilidad **99.9%** (~2h11m/mes de caída, QA-8).
- Respuesta web **p95 ≤ 800 ms** (QA-6).
- FCP móvil **< 1.4 s** (QA-7).
- Refresco de terceros **< 5 min** (QA-5).

### Alertas (3 a.m. test, EEMUA)
- Cada alerta debe ser: relevante, única, oportuna, priorizada, comprensible, diagnóstica, advisory, enfocada.
- Alertas iniciales: caída de unidad (healthz), lag de consumer > 60 s, tasa de no-match > 20%, breaker abierto > 5 min.

## 2. Backup y restore

- **PostgreSQL (Managed DO):** backups automáticos diarios (retención 7 días) + snapshot manual pre-deploy. Restore probado trimestralmente (game day).
- **Redis:** AOF (appendonly yes) + snapshot RDB cada 6 h. Redis es caché; pérdida aceptable (reconstruible).
- **NATS:** JetStream file store en volumen Docker; backup del volumen `nats_data` diario.
- **ClickHouse (Fase 1):** backup a DO Spaces (S3-compatible) semanal.
- **RPO:** 24 h (Fase 0) → 1 h (Fase 1 con WAL shipping).
- **RTO:** < 30 min (restore desde backup managed).

## 3. Retención y privacidad (FR-2, FR-3, FR-9, QA-9)

- **Emails crudos:** NO se almacenan completos. Solo se persiste el `sha256` del correo + los campos extraídos (proveedor, ref, segmentos, pasajeros). El correo original se descarta tras el parseo (Datensparsamkeit, Newman ch.11).
- **Datos personales (PII):** pasajeros (nombre, documento) se almacenan en `filius.bookings.details` (JSONB) solo mientras el viaje esté activo. Al archivar el viaje (`trip.archived`), se anonimizan los campos PII (nombre → hash, documento → null) en un job de Spiritus.
- **Monetización (FR-9):** solo insights **agregados y anónimos** (conteos por destino/periodo, tasas de cancelación). Nunca se expone PII a partners. Requiere consentimiento del usuario (términos de servicio).
- **Retención OLTP:** viajes archivados se mantienen 24 meses; luego se purgan (job mensual).
- **Retención OLAP (ClickHouse):** 24 meses de eventos crudos; agregados indefinidos.
- **Derecho de borrado (GDPR-like):** endpoint `DELETE /me/data` (Fase 2) que anonimiza todas las tablas del usuario en los 3 esquemas.

## 4. Seguridad (resumen operativo)

- **TLS:** Let's Encrypt vía Certbot, renovación automática. Nginx como único borde expuesto (ADR-105).
- **Secretos:** Fase 0 vía env del compose; Fase 2 → Vault con secretos por instancia, rotación y TTL (Newman ch.11).
- **Mínimo privilegio DB:** cada unidad conecta con un rol PG con acceso solo a su esquema (Fase 2: control de acceso por rol).
- **Cifrado en reposo:** PG managed (disk encryption), ClickHouse (Fase 1).
- **Backups cifrados:** sí (DO managed).
- **Patching:** actualizar imágenes base semanalmente; `dependabot` para Go modules.

## 5. Runbook de incidentes (Fase 0)

| Síntoma | Acción |
|---------|--------|
| Pater no responde | `docker compose restart pater`; verificar Filius up; revisar logs |
| Filius caído | `docker compose restart filius`; verificar PG/NATS; degradar Pater a "datos desactualizados" |
| NATS caído | Filius↔Spiritus se desacoplan; eventos se acumulan en JetStream (si el nodo vuelve); si el nodo se perdió, restaurar desde backup |
| PG caído | Todas las unidades fallan; esperar restore managed (RTO < 30 min) |
| Spiritus caído | Pater y Filius siguen operando; insights desactualizados; correos se acumulan en la bandeja (re-scan al volver) |

## 6. Fases

- **Fase 0:** este documento (monitoring básico + backups managed + runbook manual).
- **Fase 1:** Loki/Grafana, alertas automáticas, Redis/NATS managed, ClickHouse backup a Spaces.
- **Fase 2:** K3s + ArgoCD (GitOps), Vault, OIDC, control de acceso por rol de DB, GDPR endpoint.
