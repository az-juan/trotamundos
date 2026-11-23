```mermaid
graph LR
    CLIENT["Navegador"] -->|"HTTPS + cookie firmada"| PATER["🕊️ PATER<br/>Web BFF SSR · templ/HTMX · sqlc"]

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
    SPIRITUS -->|"read-model (referencia Fase 1); Fase 0: PG + sqlc"| PATER

    %% Externos
    FILIUS <-->|"REST resiliente (en proceso hacia out)"| EXT["Aerolíneas · Hoteles · Rentadoras"]
    SPIRITUS <-->|"OAuth2 / IMAP"| MAIL["Gmail · IMAP"]
    PATER <-->|"ACL + redaccion PII (Fase 1)"| LLM["Proveedor LLM"]
    PATER -->|"deep-link B2B (FR-6, Fase 1)"| AG["Agencia preferida"]

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
    style LLM fill:#3a2a1a,stroke:#ff9800,color:#fff
    style AG fill:#3a1a1a,stroke:#f44336,color:#fff
```

> **ADR-111 (prototipo Fase 0):** la arista REST `Pater↔Filius` del triángulo queda como **contrato de referencia**; en la ejecución real los handlers de Pater consultan PostgreSQL con **sqlc** directamente. NATS sigue solo en `Filius↔Spiritus`.


## Propósito de esta vista (component-and-connector, Bass ch.1/22)

Vista de **Componentes y Conectores (C&C)**: estructura de **runtime**. Muestra **cómo interactúan los
componentes dinámicamente** para ejecutar los escenarios del sistema. La disposición es **triangular**:
las tres unidades conectadas dos a dos, y **cada arista usa un conector distinto** (es la perspectiva en
runtime del *triángulo trinitario* de `vista_modulos.md`):

- **Pater ↔ Filius — REST (referencia Fase 1).** En Fase 0 el handler consulta PostgreSQL con sqlc (ADR-111).
  Dentro de Filius, Itinerary↔Integration se llaman **en proceso** (sin red).
- **Filius ↔ Spiritus — NATS JetStream (asíncrono, event-driven).** Propagación de hechos; cada
  consumidor reacciona por su cuenta (invertir responsabilidad, Newman ch.4).
- **Spiritus ↔ Pater — read-model / SSE (referencia Fase 1).** En Fase 0 el Rostro lee insights con sqlc;
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
    N->>EI: (Spiritus) re-publica booking.discovered si hace falta
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
    G->>G: validar cookie firmada + rate limit
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
    participant OLAP as Store Analítico (PG spiritus en Fase 0 / ClickHouse en Fase 1)
    U->>WB: GET /insights (HTTPS, JWT)
    WB->>AN: GET read-model/insights (arista Spiritus→Pater)
    AN->>OLAP: query a proyecciones pre-agregadas (ADR-107)
    OLAP-->>AN: agregados (tendencias, cohortes, FR-7)
    AN-->>WB: JSON de insights
    WB-->>U: HTML/fragmento htmx
    Note over AN,OLAP: No pasa por Filius ni por el esquema filius → la analítica no degrada el SLA de 800 ms (QA-15, ADR-107).
```

## Escenario 6 — Asistente Virtual: consulta, derivación y seguimiento (Fase 1)

```mermaid
sequenceDiagram
    participant U as Viajero (web/movil)
    participant GW as Pater · gateway
    participant AS as Pater · assistant
    participant FI as Filius · assistance/itinerary
    participant N as NATS JetStream
    participant SP as Spiritus · analytics
    participant AG as Agencia preferida (FR-6)
    U->>GW: POST /assistant/messages (JWT + X-Correlation-ID)
    GW->>AS: validar JWT + rate limit (en proceso)
    AS->>AS: IntentClassifier (LLM con ACL/PII redactada → fallback reglas)
    alt consulta de estado / incidencia menor
        AS->>FI: GET /trips/{id} o /bookings/{id} (REST, timeout)
        FI-->>AS: estado canonico (cache → PG esquema filius)
        AS-->>U: SSE/htmx: respuesta en streaming (p95 < 2 s, QA-16)
    else reprogramacion guiada
        AS-->>U: deep-link al proveedor (FR-6) + pasos
        FI-->>N: booking.confirmed (cuando el proveedor confirma)
    else caso complejo → agencia
        AS->>FI: POST /internal/assistance/cases (REST)
        FI->>FI: tx local: filius.assistance_cases (open) + token Redis TTL 15 min
        FI->>N: assistance.handoff_requested
        N->>SP: ingesta → metrica de resolucion (QA-17)
        AS-->>U: deep-link a agencia con context_token (un solo uso)
        AG->>FI: consulta contexto con token → caso handed_off
        FI->>N: assistance.resolved → SP cierra la metrica
    end
    Note over AS,FI: Sin cuarta arista: Pater solo REST; NATS sigue solo Filius↔Spiritus (ADR-103).
```

## Leyenda

- **Flecha sólida** = llamada síncrona (request-response, con timeout). · **Flecha punteada** = evento
  NATS JetStream (asíncrono; el productor publica, el consumidor es dueño de su reacción).
- Cada escenario usa **correlation-id** propagado (OTel) para trazabilidad de extremo a extremo.
- **Dentro de una unidad no hay red:** Itinerary↔Integration (Filius), gateway↔BFF (Pater),
  email-ingest↔analytics (Spiritus) se comunican por **llamada en proceso**. Solo las 3 aristas del
  triángulo cruzan procesos.
- Ver detalle por unidad en `adrs/adr-102` (módulos) y `adrs/adr-103` (conectores).
- **Asistente Virtual (Fase 1, ADR-109/110): no añade conectores.** El LLM y la agencia son actores
  externos; Pater solo habla REST con Filius y Filius publica `assistance.*` en la arista NATS ya existente.
