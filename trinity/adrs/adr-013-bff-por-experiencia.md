## ADR-013: BFF por experiencia (Web + Mobile) tras el API Gateway

> ⚠️ **Procedencia:** este ADR describe la propuesta original de microservicios (`main`). La decisión vigente para la Trinidad es ADR-101..105. Se conserva como procedencia y adaptación; no es la decisión activa.


**Status**: Aceptado
**Fecha**: 2026-10-27
**Decidido por**: Equipo de arquitectura

> **Adaptación a Trinidad (ADR-102):** el principio *one BFF per experience* se conserva, ahora como **módulos dentro de Pater** (`web-bff` + `mobile-bff`) en vez de deployables independientes; el tráfico sigue entrando por el Gateway de Pater.

### Contexto

El sistema atiende dos experiencias de cliente con necesidades de agregación distintas: la **web**
(server-rendered, HTML/htmx) y el **mobile nativo** (JSON, payloads optimizados por pantalla).
La vista de despliegue previa mostraba al Web BFF llamando **directamente** a los servicios de
dominio, saltándose el Gateway — lo que contradecía el ADR-006 ("punto de entrada único para Web y
Mobile") y dejaba fuera del gateway el rate limiting, tracing y auth de esas llamadas.

### Decisión

Cada experiencia tiene **su propio BFF** (patrón *one BFF per experience*, Newman ch.14) y **todo el
tráfico de clientes entra por el API Gateway** (single entry):

- **Web BFF** (`templ + htmx + Tailwind`, HTML + SSE) — deployable propio, puerto `:3000`.
- **Mobile BFF** (`chi`, API JSON agregada) — deployable propio, puerto `:3001`.

Flujo: `cliente → Nginx (TLS) → API Gateway → BFF de su experiencia → servicios de dominio` (red
interna). El Gateway aplica auth, rate limit y tracing a **todas** las peticiones; los BFF agregan
datos de Itinerary/Integration para componer su vista y **no** exponen sus llamadas internas a
internet.

| Decisión | Justificación |
|----------|---------------|
| Un BFF por experiencia | Cada uno optimiza su payload (HTML vs JSON) y evoluciona de forma independiente |
| Todo pasa por el Gateway | Rate limit, auth y tracing uniformes; los BFF no son alcanzables desde fuera |
| BFF llama a servicios por red interna | Sin duplicar auth en el borde; el Gateway ya autorizó al cliente |

### Consecuencias

| Positivas | Negativas / Riesgos |
|-----------|---------------------|
| Cada experiencia evoluciona sin afectar a la otra (independencia de deploy) | Un deployable más (7 en Fase 0): mitigado con template de CI/CD (ADR-001) |
| El Gateway es la frontera de seguridad y observabilidad única | Los BFF añaden un salto de red interno (Go→Go, < 1 ms) |
| Mobile recibe JSON ligero (mejora FCP < 1.4 s, QA-7) sin arrastrar HTML | Duplica la lógica de agregación; se comparte vía `shared/` y clientes generados desde OpenAPI |
| Resuelve la inconsistencia de la vista de despliegue previa (BFF saltando el Gateway) | |

### Alternativas descartadas

- **BFF único compartido**: acopla las dos experiencias y obliga a un payload que sirve a medias a ambas.
- **BFF llamando directo a servicios (sin Gateway)**: rompía el "single entry" y dejaba sin rate limit/auth esas llamadas.
