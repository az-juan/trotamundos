```mermaid
graph TD
    TITLE["Vista de Módulos — Estructura de código (monorepo · servicios desacoplados)"]

    ROOT["trotamundos/ — monorepo<br/>go.work (workspace Go)"]

    ROOT --> CMD["cmd/ — main de cada deployable"]
    ROOT --> SVC["services/ — un módulo Go independiente por microservicio<br/>(go.mod propio ⇒ build y deploy independientes)"]
    ROOT --> SH["shared/ — contratos y andamiaje compartidos<br/>(solo lo estable: eventos, clientes, config)"]
    ROOT --> DEP["deploy/ — docker-compose · nginx · k8s"]
    ROOT --> DOC["docs/ — openapi.yaml · events.yaml (SDD)"]

    CMD --> C1["gateway/main.go"]
    CMD --> C2["web-bff/main.go"]
    CMD --> C3["mobile-bff/main.go"]

    SVC --> S1["services/itinerary/ (go.mod)"]
    SVC --> S2["services/email-ingest/ (go.mod)"]
    SVC --> S3["services/integration/ (go.mod)"]
    SVC --> S4["services/analytics/ (go.mod)"]

    S1 --> D1["internal/domain<br/>(ENTIDADES · 0 imports externos)"]
    S1 --> U1["internal/usecase<br/>(orquestación)"]
    S1 --> P1["internal/ports<br/>(interfaces Go)"]
    S1 --> A1["internal/adapters/in · out"]

    SH --> H1["event · resilient<br/>otelsetup · config · idgen"]

    classDef leaf fill:#1a2a3a,stroke:#64b5f6,color:#fff;
    classDef mod fill:#2a1a3a,stroke:#ba68c8,color:#fff;
    class C1,C2,C3,S1,S2,S3,S4 mod;
    class D1,U1,P1,A1,H1 leaf;
```

## Propósito de esta vista (module structure, Bass ch.1/22)

Vista de **módulos** (estructura estática, tiempo de diseño): muestra **cómo se organiza el código**
para razonar sobre **modificabilidad**, build y asignación de trabajo. El mensaje central es el
**desacoplamiento**: cada microservicio es un **módulo independiente** (su propio `go.mod`) que se
compila y despliega solo (independent deployability, Newman ch.1), y dentro de él la **arquitectura
hexagonal** aísla la lógica de negocio de la infraestructura.

## 1. Monorepo y límites de módulo

- **Un módulo Go por microservicio** (`services/<nombre>/go.mod`). El `go.work` del raíz los reúne en
  un workspace para construirlos juntas, pero **cada uno sigue siendo una unidad de build/deploy
  independiente** — la fuerza motriz del desacoplamiento.
- **`shared/` solo contiene lo estable**: tipos de eventos, el cliente resiliente, setup de OTel,
  config e IDs. **Nada de lógica de dominio compartida** → si un concepto empieza a compartirse, es
  señal de que debería ser un servicio (Newman ch.5, DRY perils).
- **Sin base de datos compartida** (Newman ch.2, anti-pattern #1): cada servicio es dueño de sus
  tablas; las conexiones entre módulos son contratos (puertos/eventos), no suposiciones sobre el
  interior del otro.

## 2. Estructura hexagonal interna (un servicio típico)

```mermaid
graph BT
    subgraph "services/itinerary/internal/ — dependencias apuntan HACIA ADENTRO"
        AIN["adapters/in<br/>http (chi) · nats consumer · cli"]
        AOUT["adapters/out<br/>pgx · go-redis · nats producer · resilient"]
        PORTS["ports<br/>BookingRepository · EventPublisher · CacheStore (interfaces)"]
        UC["usecase<br/>CreateTrip · RegisterDiscovered · ArchiveCompletedTrips"]
        DOM["domain<br/>Trip · Booking · Traveler · state machine<br/>(pure Go — NO importa pgx/nats.go/net/http)"]
        AIN --> UC
        AOUT --> PORTS
        UC --> PORTS
        UC --> DOM
        DOM -. reglas de negocio .-> PORTS
    end
    classDef port fill:#3a2a1a,stroke:#ff9800,color:#fff;
    classDef dom fill:#1a3a2a,stroke:#81c784,color:#fff;
    class PORTS port;
    class DOM dom;
```

**Regla de desacoplamiento (information hiding, Parnas/Newman ch.1)**: el **dominio no conoce
infraestructura**. Cambiar PostgreSQL→OtraDB, NATS→Kafka o Gmail→Outlook es reemplazar un adaptador
(`adapters/out`) sin tocar `domain` ni `usecase`. Los **puertos** son interfaces que el dominio
define y los adaptadores implementan; el flujo de dependencias va siempre **hacia el centro** (hacia
dominio/puertos), nunca hacia afuera.

## 3. Mecanismos de desacoplamiento (por qué los servicios no se ataran)

| Mecanismo | Qué logra | Referencia |
|-----------|-----------|-----------|
| `go.mod` por servicio | Build/deploy independiente → puedo cambiar Itinerary sin reconstruir los demás | Newman ch.1 |
| Hexagonal (puertos/adaptadores) | Dominio estable; infraestructura intercambiable | ADR-002 |
| Comunicación por eventos (NATS) | Los servicios colaboran sin conocerse (invertir responsabilidad) | ADR-005, Newman ch.4 |
| `shared/` solo contratos | Evita acoplamiento de contenido; se comparte la interfaz, no la implementación | Newman ch.5 |
| Sin DB compartida | Cada servicio es dueño de sus datos; nadie escribe en el interior del otro | Newman ch.2 |
| Contratos generados (OpenAPI/JSON-Schema) | API y eventos versionados; expansion/contract en vez de breaking changes | ADR-006, Newman ch.5 |

## 4. Leyenda

- **azul** = adaptadores (infraestructura, cambia a menudo) · **verde** = dominio (estable) ·
  **naranja** = puertos (interfaces, el contrato) · **morado** = módulos/módulos-raíz · **gris** = árbol de carpetas.
- **Flechas**: dependencia de compilación (apunta hacia el módulo del que se depende). En el hexágono,
  todas apuntan hacia `domain`/`ports` = el centro estable.

> Las vistas complementarias: `vista_cyc.md` (interacción dinámica en runtime) y `vista_despliegue.md`
> (asignación a infraestructura). Detalle por servicio en `microservices/01..07`.
