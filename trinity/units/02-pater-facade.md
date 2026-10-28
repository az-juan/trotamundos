# 02 — Unidad Pater (el Rostro) — Gateway · Web BFF · Mobile BFF

> **Rol**: presentación y entrada (facade). Es la **única** unidad expuesta al borde (atrás de Nginx/TLS).
> **Perfil de carga**: per-request, baja latencia (web ≤ 800 ms, QA-6; FCP móvil < 1.4 s, QA-7).
> **ADR de origen**: 102 (unidad), 103 (arista REST con Filius). Complementa: 105 (despliegue), 104 (JWT).
> **Colapsa** los deployables de `main`: API Gateway (ADR-006) + Web BFF (ADR-009) + Mobile BFF (ADR-013),
> ahora **módulos internos en proceso** dentro de `units/pater`.

## 1. Responsabilidad

Pater es el **Rostro** del sistema: recibe toda entrada (web, móvil, terceros) y la agrega/renderiza.
No contiene dominio operacional; delega lecturas/escrituras en **Filius** por REST (arista Pater↔Filius).

- **gateway/** — punto único de entrada (ADR-013): routing, **validación JWT RS256** (JWKS, ADR-014),
  rate limit por cliente (token bucket, `golang.org/x/time/rate`), proxy inverso hacia Filius,
  inyección de `X-Correlation-ID`. Generado con `oapi-codegen` desde `docs/api/openapi.yaml`.
- **web-bff/** — render del servidor con **templ + htmx + Tailwind** (ADR-009): HTML completo en la
  primera respuesta (sin hydration), swap de fragmentos vía htmx, SSE para notificaciones en vivo
  (arista Spiritus→Pater). i18n por región.
- **mobile-bff/** — agregación **JSON** para las apps nativas (Swift/Kotlin) con Chi (ADR-013):
  *one BFF per experience*. Mismos contratos OpenAPI que la web, payload adaptado a móvil.

## 2. Tecnologías

| Capa | Tecnología | Uso en Pater |
|------|-----------|--------------|
| Lenguaje | Go 1.23+ | lógica de facade |
| HTTP server / router | `github.com/go-chi/chi/v5` | routing del gateway y del mobile-bff |
| Codegen | `oapi-codegen` | tipos + rutas desde `openapi.yaml` |
| Templates web | `github.com/a-h/templ` | templates compilados a `.go` (web-bff) |
| Interactividad | htmx 1.9+ | swap de fragmentos, SSE, polling |
| CSS | Tailwind CSS 3.x | utilities → build estático |
| Auth | JWT RS256 (JWKS) | validación de token (Fase 0; emisión por el Gateway, ADR-014) |
| Rate limit | `golang.org/x/time/rate` | token bucket por cliente/API key |
| Cache | `github.com/redis/go-redis/v9` | sesiones, rate limit, plantillas renderizadas |
| Tracing/metrics | OpenTelemetry + Prometheus | latencia p99, tasa de error, rate-limit hits |

## 3. Estructura (vista de módulos, hexagonal por módulo)

```mermaid
graph TB
    subgraph "units/pater/internal/ — gateway · web-bff · mobile-bff (en proceso)"
        IN["Adaptadores de entrada<br/>• HTTP (chi): gateway, web-bff, mobile-bff<br/>• SSE publisher (notificaciones)"]
        PORTSIN["Puertos (entrada)<br/>• APIPort (queries)"]
        UC["Use Cases<br/>• Authenticate<br/>• RateLimit<br/>• RenderTripPage / RenderInsights<br/>• AggregateMobileTrip"]
        PORTSOUT["Puertos (salida)<br/>• FiliusClient (REST)<br/>• ReadModelClient (Spiritus)<br/>• CacheStore"]
        OUT["Adaptadores de salida<br/>• chi (HTTP out)<br/>• go-redis → Redis<br/>• http (REST a Filius, timeout)<br/>• SSE (a web-bff)"]
        IN --> PORTSIN --> UC
        UC --> PORTSOUT
        OUT -. implements .-> PORTSOUT
    end
    style UC fill:#1a3a2a,stroke:#81c784,color:#fff
```

> **Sin dominio propio**: los "use cases" de Pater orquestan **llamadas a Filius** (REST) y **lecturas
> de read-model** (Spiritus). La regla hexagonal (ADR-002) se respeta: el *FiliusClient* es un puerto
> (interfaz) que el adaptador `http` implementa → cambiar el transporte no toca la lógica de agregación.

## 4. Interacción con las otras unidades (aristas del triángulo)

- **Pater → Filius (REST síncrono)**: el gateway proxya / el web-bff y mobile-bff llaman a los endpoints
  de Filius (`GET /bookings/{id}`, `GET /trips/{id}`, comandos de reserva). Siempre con **timeout** de
  contexto y el `X-Correlation-ID` propagado. Si Filius no responde, Pater degrada a "datos
  desactualizados" (no bloquea el render, ADR-103).
- **Spiritus → Pater (read-model / SSE)**: el web-bff consume la API de insights de Spiritus para el
  dashboard (sin pasar por Filius/OLTP, QA-15) y suscribe a **SSE** para empujar notificaciones en vivo
  al navegador (disrupciones, `trip.disrupted`).

```mermaid
sequenceDiagram
    participant U as Browser/App
    participant G as Pater · gateway
    participant WB as Pater · web-bff
    participant FI as Filius (itinerary)
    U->>G: GET /trips/123 (HTTPS, JWT)
    G->>G: validar JWT RS256 + rate limit
    G->>WB: route (X-Correlation-ID) (en proceso dentro de Pater)
    WB->>FI: GET /bookings?trip=123 (REST, timeout)
    FI-->>WB: trip + bookings
    WB-->>U: HTML completo (FCP < 1.4 s, sin hydration)
```

## 5. Trazabilidad de drivers

| Driver | Cómo se satisface en Pater |
|--------|----------------------------|
| QA-6 (web ≤ 800 ms) | REST directo a Filius, sin capa intermedia Go→Go innecesaria; cache Redis de plantillas |
| QA-7 (FCP móvil < 1.4 s) | web-bff SSR (HTML en la primera respuesta), mobile-bff JSON ligero |
| QA-9 (acceso seguro) | validación JWT RS256 en el gateway; secretos por instancia |
| QA-12 (trazabilidad) | `X-Correlation-ID` inyectado y propagado por OTel a Filius/Spiritus |
| QA-14 (canales web/móvil) | web-bff (HTML/htmx) y mobile-bff (JSON) como experiencias separadas (ADR-013) |

> Vistas complementarias: `../architecture/vista_cyc.md` (escenarios 2 y 3 en runtime) y
> `../architecture/vista_despliegue.md` (contenedor `pater` tras Nginx). Detalle de eventos en
> [05-catalogo-eventos.md](./05-catalogo-eventos.md).
