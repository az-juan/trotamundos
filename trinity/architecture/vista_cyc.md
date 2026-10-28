```mermaid
graph LR
    CLIENT["Web / Mobile / Terceros"] -->|"REST (JWT RS256)"| PATER["🕊️ PATER<br/>Gateway · Web BFF · Mobile BFF"]

    PATER -->|"REST interno (en proceso)"| WEB["web-bff<br/>HTML/htmx"]
    PATER -->|"REST interno (en proceso)"| MOB["mobile-bff<br/>JSON"]
    PATER -->|"REST (dentro de Filius, en proceso)"| FILIUS["✝️ FILIUS<br/>Itinerary · Integration"]

    WEB -->|"llama a módulos de Filius"| FILIUS
    MOB -->|"llama a módulos de Filius"| FILIUS

    %% Triángulo: las tres aristas
    FILIUS -. "booking.confirmed · trip.*" .-> NATS["NATS JetStream<br/>(arista async)"]
    NATS -. "booking.discovered · trip.disrupted" .-> SPIRITUS["🔥 SPIRITUS<br/>Email Ingest · Analytics"]
    SPIRITUS -. "consume / re-publica" .-> NATS

    %% Arista Spiritus -> Pater (read-model / SSE)
    SPIRITUS -->|"read-model (insights) · SSE (notificaciones)"| PATER

    %% Externos
    FILIUS <-->|"REST resiliente (en proceso hacia out)"| EXT["Aerolíneas · Hoteles · Rentadoras"]
    SPIRITUS <-->|"OAuth2 / IMAP"| MAIL["Gmail · IMAP"]

    %% Datos
    FILIUS -->|"SQL (esquema filius)"| PG[(PostgreSQL<br/>OLTP · compartido)]
    SPIRITUS -->|"SQL (esquema spiritus)"| PG
    PATER -->|"cache (sesiones/rate-limit)"| REDIS[(Redis 7<br/>compartido)]
    FILIUS -->|"cache"| REDIS
    SPIRITUS -->|"SQL batch"| CH[(ClickHouse<br/>OLAP · de Spiritus)]

    style PATER fill:#2d2d44,stroke:#7c6ff7,color:#fff
    style FILIUS fill:#1a3a2a,stroke:#4caf50,color:#fff
    style SPIRITUS fill:#2a1a3a,stroke:#ba68c8,color:#fff
    style NATS fill:#1a2a3a,stroke:#2196f3,color:#fff
```

## Propósito de esta vista (component-and-connector, Bass ch.1/22)

Vista de **Componentes y Conectores (C&C)**: estructura de **runtime**. Muestra **cómo interactúan los
componentes dinámicamente** para ejecutar los escenarios del sistema. La disposición es **triangular**:
las tres unidades conectadas dos a dos, y **cada arista usa un conector distinto** (es la perspectiva en
runtime del *triángulo trinitario* de `vista_modulos.md`):

- **Pater ↔ Filius — REST (síncrono, request-response).** Cliente → Gateway → BFF → módulos de Filius.
  Dentro de Filius, Itinerary↔Integration se llaman **en proceso** (sin red).
- **Filius ↔ Spiritus — NATS JetStream (asíncrono, event-driven).** Propagación de hechos; cada
  consumidor reacciona por su cuenta (invertir responsabilidad, Newman ch.4).
- **Spiritus ↔ Pater — read-model / SSE.** El Rostro lee proyecciones analíticas y empuja
  notificaciones; **sin pasar por el Núcleo OLTP** (CQRS; protege QA-15).

> **Arista inexistente a propósito:** no hay NATS entre Pater y el cliente, ni REST síncrono
> Filius→Spiritus para propagar hechos. La Trinidad **no duplica conectores**: cada par de unidades
> habla exactamente un idioma, el que la interacción exige.

## Escenario 1 — Reserva descubierta por email (flujo event-driven)

```mermaid
sequenceDiagram
    participant M as Mail (Gmail/IMAP)
    participant EI as Spiritus · email-ingest
    participant N as NATS JetStream
    participant IT as Filius · itinerary
    participant IN as Filius · integration
    M->>EI: correo nuevo (IDLE / webhook)
    EI->>EI: dedup SHA256 + whitelist (en proceso)
    EI->>EI: parse (regex → template)
    EI->>N: publish booking.discovered
    N->>IT: booking.discovered
    IT->>IT: crea/actualiza booking (tx ACID local, en proceso)
    IT->>IN: llamada en proceso (itinerary→integration)
    IN->>IN: fetch detalle al proveedor (resiliente)
    IN->>N: publish booking.confirmed (con detalle)
    N->>EI: (Spiritus) re-ingesta si hace falta
    Note over IT,EI: Sin tx distribuida (Newman ch.6).<br/>Itinerary e Integration son módulos de una misma unidad (Filius): la orquestación entre ellos es una llamada en proceso.
```

## Escenario 2 — Usuario ve su itinerario en la web (síncrono)

```mermaid
sequenceDiagram
    participant U as Browser
    participant NG as Nginx (TLS)
    participant G as Pater · gateway
    participant WB as Pater · web-bff
    participant IT as Filius · itinerary
    participant R as Redis / PostgreSQL
    U->>NG: GET /trips/123 (HTTPS)
    NG->>G: proxy
    G->>G: validar JWT RS256 (JWKS) + rate limit
    G->>WB: route (X-Correlation-ID) (en proceso dentro de Pater)
    WB->>IT: GET booking/trip 123 (REST a Filius)
    IT->>R: lectura (cache → PG)
    R-->>IT: trip + bookings
    IT-->>WB: vista
    WB->>WB: render templ + i18n (en proceso)
    WB-->>U: HTML completo (FCP < 1.4 s, sin hydration)
```

## Escenario 3 — Disrupción de proveedor (webhook → notificación al usuario)

```mermaid
sequenceDiagram
    participant P as Proveedor
    participant IN as Filius · integration
    participant N as NATS JetStream
    participant IT as Filius · itinerary
    participant AN as Spiritus · analytics
    participant WB as Pater · web-bff
    P->>IN: POST /webhooks/{provider} (X-Signature)
    IN->>IN: verificar firma HMAC + idempotencia (Redis)
    IN->>IN: mapear a booking canónico
    IN->>N: publish trip.disrupted
    N->>IT: actualizar estado del booking/trip
    N->>AN: ingesta (batch) a ClickHouse
    N->>WB: SSE empuja el evento al navegador (arista Spiritus→Pater)
    IT->>N: (si aplica) publish booking.cancelled
```

## Escenario 4 — Llamada resiliente a proveedor (cadena de protección, dentro de Filius)

```mermaid
sequenceDiagram
    participant U as Origen (BFF / webhook / cron)
    participant INT as Filius · integration
    participant CB as CircuitBreaker
    participant RT as Retry + RateLimiter
    participant H as HTTP Client (timeout)
    participant EXT as Proveedor
    U->>INT: request (context timeout)
    INT->>CB: llamar por proveedor
    alt breaker abierto (≥5 fallos)
        CB-->>U: fallback → "datos desactualizados"
    else cerrado
        CB->>RT: retry (429/5xx/timeout, exp+jitter)
        RT->>H: rate limit (token bucket)
        H->>EXT: GET /booking/{ref}
        alt respuesta ok
            EXT-->>U: datos → mapeo a canónico
        else fallos repetidos
            H-->>CB: fallo → abrir breaker (30 s)
        end
    end
```

## Escenario 5 — Dashboard de insights (lectura analítica sin tocar el OLTP)

```mermaid
sequenceDiagram
    participant U as Browser
    participant WB as Pater · web-bff
    participant AN as Spiritus · analytics
    participant CH as ClickHouse
    U->>WB: GET /insights (HTTPS, JWT)
    WB->>AN: GET read-model/insights (arista Spiritus→Pater)
    AN->>CH: query a vista materializada (≤ 30 s, batch)
    CH-->>AN: agregados (tendencias, cohortes, FR-7)
    AN-->>WB: JSON de insights
    WB-->>U: HTML/fragmento htmx
    Note over AN,CH: No pasa por Filius/PostgreSQL → la analítica pesada no degrada el SLA de 800 ms (QA-15).
```

## Leyenda

- **Flecha sólida** = llamada síncrona (request-response, con timeout). · **Flecha punteada** = evento
  NATS JetStream (asíncrono; el productor publica, el consumidor es dueño de su reacción).
- Cada escenario usa **correlation-id** propagado (OTel) para trazabilidad de extremo a extremo.
- **Dentro de una unidad no hay red:** Itinerary↔Integration (Filius), gateway↔BFF (Pater),
  email-ingest↔analytics (Spiritus) se comunican por **llamada en proceso**. Solo las 3 aristas del
  triángulo cruzan procesos.
- Ver detalle por unidad en `adrs/adr-102` (módulos) y `adrs/adr-103` (conectores).
