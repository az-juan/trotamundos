## ADR-008: Procesamiento de Correo — IMAP + Gmail API + Parser por Reglas

> ⚠️ **Procedencia:** este ADR describe la propuesta original de microservicios (`main`). La decisión vigente para la Trinidad es ADR-101..105. Se conserva como procedencia y adaptación; no es la decisión activa.


**Status**: Aceptado
**Fecha**: 2026-10-26
**Decidido por**: Equipo de arquitectura

> **Adaptación a Trinidad (ADR-102):** el procesamiento de correo se conserva como **módulo `email-ingest` dentro de la unidad Spiritus** (perfil batch/async). Publica `booking.discovered` en la arista con Filius.

### Contexto

El sistema debe escanear el correo de los usuarios para detectar reservas automáticamente. Los usuarios usan Gmail, Outlook, IMAP genérico. El formato de los correos de confirmación es no estructurado y varía por proveedor.

### Decisión

| Capa | Tecnología |
|------|-----------|
| Conexión Gmail | `google.golang.org/api/gmail/v1` (push via Pub/Sub, labels) |
| Conexión IMAP genérico | `emersion/go-imap/v2` (IDLE para push, polling 60 s como fallback) |
| Auth | OAuth2 (Gmail) / XOAUTH2 (IMAP) / Basic (fallback, desaconsejado) |
| Parsing MIME | `net/mail` (stdlib) |
| Extracción de datos | **Fase 0**: regex + tablas de patterns por proveedor. **Fase 1**: templates por proveedor. **Fase 2**: LLM API para el 5–10% no matcheado. |
| Deduplicación | SHA256(From + Subject + Date) en Redis, TTL 7 días |
| Whitelist | Tabla PostgreSQL + cache en Redis |
| Dead letter | NATS JetStream stream con retención 7 días |

### Consecuencias

| Positivas | Negativas / Riesgos |
|-----------|-------------------|
| OAuth2: acceso seguro, revocable, sin contraseñas en claro | Dependencia de Google API (rate limit: 250 req/min por usuario) |
| IDLE: push en tiempo real, no polling agresivo | IMAP genérico: no todos los proveedores soportan IDLE (mitigado: fallback a polling) |
| Regex + templates: determinístico, testeable, sin costo por call | El parser necesita mantenimiento cuando un proveedor cambia su template (mitigado: dead letter + alertas) |
| Dead letter: ningún correo se pierde | |

### Criterio de adopción de LLM

Agregar llamada a LLM (Claude/GPT) solo si:
1. La tasa de "no matcheado" supera el 10% de correos procesados.
2. El costo por call (< $0.01) es aceptable vs. el costo de no detectar la reserva.

