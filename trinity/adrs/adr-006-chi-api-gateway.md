## ADR-006: Chi como API Gateway (DIY)

> ⚠️ **Procedencia:** este ADR describe la propuesta original de microservicios (`main`). La decisión vigente para la Trinidad es ADR-101..105. Se conserva como procedencia y adaptación; no es la decisión activa.


**Status**: Aceptado
**Fecha**: 2026-10-26
**Decidido por**: Equipo de arquitectura

> **Adaptación a Trinidad (ADR-102):** Chi se conserva como framework del **módulo `gateway` dentro de la unidad Pater** (ya no es un deployable independiente).

### Contexto

Se necesita un punto de entrada único para Web y Mobile que maneje routing, autenticación (JWT), rate limiting y reverse proxy hacia los servicios internos. Se evaluaron: Chi (DIY), Traefik, Caddy, Tyk, Envoy Gateway, Ambassador, Kong.

### Decisión

**Chi** (`github.com/go-chi/chi/v5`) como framework del gateway, con:

- `oapi-codegen` para generar rutas y types desde `openapi.yaml` (spec-driven).
- `httputil.ReverseProxy` para el proxy hacia servicios internos.
- Middleware propios: JWT validation, rate limit (Redis), tracing (OpenTelemetry).
- En Fase 2 (K8s): migrar a **Envoy Gateway** (control plane Go, data plane Envoy) sin tocar los servicios.

### Consecuencias

| Positivas | Negativas / Riesgos |
|-----------|-------------------|
| Cero costo infra extra (corre en el mismo Droplet) | No tiene dashboard de out-of-the-box (mitigado: OpenTelemetry + Grafana) |
| Máximo control sobre auth, rate limit, canary routing | El equipo debe mantener el gateway (mitigado: es ~300 LOC) |
| Sin lock-in: es código Go puro, migrable a cualquier infra | No tiene auto-TLS (mitigado: Nginx o Caddy como TLS terminator en Fase 0) |
| SLA de 800 ms: sin capa intermedia (Go → Go) | |

