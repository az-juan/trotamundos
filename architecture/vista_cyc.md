```mermaid
graph LR
    CLIENT["Web / Mobile / Terceros"] -->|"REST (JWT RS256)"| GW["API Gateway"]
    GW -->|"REST"| WEBBFF["Web BFF<br/>HTML/htmx"]
    GW -->|"REST"| MOBBFF["Mobile BFF<br/>JSON"]
    WEBBFF -->|"REST interno"| IT["Itinerary"]
    WEBBFF -->|"REST interno"| IN["Integration"]
    MOBBFF -->|"REST interno"| IT
    MOBBFF -->|"REST interno"| IN
    EI["Email Ingest"] -. "booking.discovered" .-> NATS["NATS JetStream<br/>Event Bus"]
    IT -. "booking.confirmed · trip.*" .-> NATS
    IN -. "booking.confirmed · trip.disrupted" .-> NATS
    NATS -. "consume" .-> IT
    NATS -. "consume" .-> AN["Analytics"]
    NATS -. "consume" .-> NT["Notification<br/>(Fase 1)"]
    IN <-->|"REST resiliente"| EXT["Aerolíneas · Hoteles · Rentadoras"]
    EI <-->|"OAuth2 / IMAP"| MAIL["Gmail · IMAP"]
    IT -->|"SQL"| PG[("PostgreSQL")]
    IT -->|"cache"| RD[("Redis")]
    AN -->|"SQL batch"| CH[("ClickHouse")]
    style GW fill:#2d2d44,stroke:#7c6ff7,color:#fff
    style NATS fill:#1a3a2a,stroke:#4caf50,color:#fff
```

## Propósito de esta vista (component-and-connector, Bass ch.1/22)

Vista de **Componentes y Conectores (C&C)**: estructura de **runtime**. Muestra **cómo interactúan los
componentes dinámicamente** para ejecutar los escenarios del sistema. Conectores usados:

- **REST** (síncrono, request-response): cliente→Gateway→BFF→servicios de dominio.
- **NATS JetStream** (asíncrono, event-driven): propagación de hechos; cada consumidor reacciona por
  su cuenta (invertir responsabilidad, Newman ch.4).
- **SQL** (servicio→su base de datos) y **REST externo resiliente** (Integration→proveedores).

## Escenario 1 — Reserva descubierta por email (flujo event-driven)

```mermaid
sequenceDiagram
    participant M as Mail (Gmail/IMAP)
    participant EI as Email Ingest
    participant N as NATS JetStream
    participant IT as Itinerary
    participant IN as Integration
    participant AN as Analytics
    M->>EI: correo nuevo (IDLE / webhook)
    EI->>EI: dedup SHA256 + whitelist
    EI->>EI: parse (regex → template)
    EI->>N: publish booking.discovered
    N->>IT: booking.discovered
    N->>IN: booking.discovered
    N->>AN: booking.discovered
    IT->>IT: crea/actualiza booking (tx ACID local)
    IN->>IN: fetch detalle al proveedor (resiliente)
    IT->>N: publish booking.confirmed
    IN->>N: publish booking.confirmed (con detalle)
    Note over IT,AN: Cada servicio reacciona de forma independiente; sin tx distribuida (Newman ch.6)
```

## Escenario 2 — Usuario ve su itinerario en la web (síncrono)

```mermaid
sequenceDiagram
    participant U as Browser
    participant NG as Nginx (TLS)
    participant GW as API Gateway
    participant WB as Web BFF
    participant IT as Itinerary
    participant R as Redis / PostgreSQL
    U->>NG: GET /trips/123 (HTTPS)
    NG->>GW: proxy
    GW->>GW: validar JWT RS256 (JWKS) + rate limit
    GW->>WB: route (X-Correlation-ID)
    WB->>IT: GET booking/trip 123
    IT->>R: lectura (cache → PG)
    R-->>IT: trip + bookings
    IT-->>WB: vista
    WB->>WB: render templ + i18n
    WB-->>U: HTML completo (FCP < 1.4 s, sin hydration)
```

## Escenario 3 — Disrupción de proveedor (webhook → notificación)

```mermaid
sequenceDiagram
    participant P as Proveedor
    participant IN as Integration
    participant N as NATS JetStream
    participant IT as Itinerary
    participant NT as Notification (Fase 1)
    participant AN as Analytics
    P->>IN: POST /webhooks/{provider} (X-Signature)
    IN->>IN: verificar firma HMAC + idempotencia (Redis)
    IN->>IN: mapear a booking canónico
    IN->>N: publish trip.disrupted
    N->>IT: actualizar estado del booking/trip
    N->>NT: avisar al usuario (push/email)
    N->>AN: ingesta (batch)
    IT->>N: (si aplica) publish booking.cancelled
```

## Escenario 4 — Llamada resiliente a proveedor (cadena de protección)

```mermaid
sequenceDiagram
    participant U as Origen (BFF / webhook / cron)
    participant INT as Integration
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

## Leyenda

- **Flecha sólida** = llamada síncrona (request-response, con timeout). · **Flecha punteada** = evento
  NATS JetStream (asíncrono; el productor publica, el consumidor es dueño de su reacción).
- Cada escenario usa **correlation-id** propagado (OTel) para trazabilidad de extremo a extremo.
- Los servicios **no comparten base de datos**; solo se comunican por REST (síncrono) o eventos
  (asíncrono). Ver detalle por servicio en `microservices/02..05`.
