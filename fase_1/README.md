# Trotamundos — Inicio Parcial de Fase 1 (Asistente Virtual MVP)

> **Estado:** Activo (Inicio parcial de Fase 1 por Disparador 5 de [`trinity/registers/transiciones.md`](../trinity/registers/transiciones.md)).  
> **Decisiones rectoras:** [ADR-109](../trinity/adrs/adr-109-asistente-virtual.md) · [ADR-110](../trinity/adrs/adr-110-derivacion-y-estado-asistente.md) · [ADR-113](../trinity/adrs/adr-113-inicio-parcial-fase1-asistente-determinista.md).

## 1. Motivación y Alcance del Inicio Parcial de Fase 1

En cumplimiento de la especificación rectora ([`skills/SPEC_ARCHITECTURE.md`](../skills/SPEC_ARCHITECTURE.md) — **Regla 1: No adelantarse**, **Regla 2: Coherencia** y **Regla 3: Sobriedad Operativa / YAGNI**), el equipo acordó que el **Asistente Virtual (`FR-10`, `FR-11`, `QA-16`, `QA-17`)** se implementaría al iniciar la **Fase 1** (manteniendo la Fase 0 libre de chat, según `trinity/drivers/design-purpose.md` §5).

Para abrir la **Fase 1** de forma incremental y disciplinada:
- **Qué se activa en esta entrega (Fase 1 parcial):**
  1. **Módulo `assistant/` en Pater (`units/pater/internal/assistant/`):** interfaz web SSR + HTMX en `/assistant` (*Mostrador de Asistencia y Telégrafo*), puerto hexagonal `IntentClassifier` con adaptador **puramente determinista** (`RuleBasedClassifier`, sin dependencia ni costo de LLM externo), redacción automática de PII (`RedactPII`, `QA-9`) y sesión efímera de conversación en **Redis** (TTL 30 min, sin persistir transcripción en PostgreSQL).
  2. **Módulo `assistance/` en Filius (`units/filius/internal/api/assistance.go`):** apertura y ciclo de vida del agregado `AssistanceCase` (`open → handed_off → resolved`) en `filius.assistance_cases`, emisión del `context_token` opaco de un solo uso en **Redis** (TTL 15 min) con *deep-link* hacia la agencia preferida (`preferred_agency_id`, `FR-6` / `FR-11`), invocado desde Pater por la arista REST (`POST /internal/assistance/cases`).
  3. **Suscripción `assistance.>` en Spiritus (`units/spiritus/cmd/spiritus/main.go`):** consumo asíncrono por **NATS JetStream** de `assistance.handoff_requested` y `assistance.resolved` para medir la tasa de resolución de primer nivel (`QA-17`).
- **Qué NO se adelanta en este incremento (permanece pendiente de sus propios disparadores):**
  - 2.º Droplet con **ClickHouse + Metabase** (`DT-01`, disparado solo por volumen analítico).
  - **Mobile BFF JSON + JWT RS256/OIDC** (`DT-08`, disparado solo al iniciar apps móviles nativas).
  - Proveedor de **LLM externo** (`DT-06`, pospuesto deliberadamente al operar con el clasificador determinista).
  - **Sin duplicación de código ni de infraestructura:** este directorio `fase_1/` contiene únicamente los artefactos incrementales y pruebas E2E propios del inicio parcial de Fase 1 (`tests/hurl/assistant.hurl`), reutilizando el despliegue de [`fase_0/deploy/docker-compose.yml`](../fase_0/deploy/docker-compose.yml).

## 2. Intenciones Soportadas por el Clasificador Determinista

| Intención (`Intent`) | Nivel | Comportamiento en el MVP |
|---|---|---|
| `status_query` | Nivel 1 (autónomo) | Consulta viajes y reservas reales del usuario con `sqlc` y emite `assistance.resolved` (`tier1_resolved: true`). |
| `reschedule_guidance` | Nivel 1 (autónomo) | Respeta la regla de oro de ADR-109 (no muta en el proveedor); entrega el *deep-link* (`FR-6`) de reprogramación. |
| `minor_issue` | Nivel 1 (autónomo) | Responde dudas operativas frecuentes (equipaje, embarque, check-in) y registra resolución Nivel 1 (`QA-17`). |
| `complex_handoff` | Nivel 2 (derivación `FR-11`) | Crea `filius.assistance_cases`, llama a Filius (`POST /internal/assistance/cases`), genera `context_token` (TTL 15 min en Redis), publica `assistance.handoff_requested` en NATS y entrega el *deep-link* a la agencia preferida. |
| `out_of_scope` | Nivel 1 (orientación) | Guía al viajero sobre las capacidades disponibles del mostrador de asistencia. |

## 3. Verificación

```bash
./scripts/test.sh   # Tests unitarios de los 4 módulos (incluye assistant_test.go y handlers_test.go)
./scripts/run.sh    # Levanta postgres, nats y redis + Filius (:8081), Spiritus (:8082) y Pater (:8090)
./scripts/hurl.sh   # Ejecuta las suites E2E de fase_0/tests/hurl/ y fase_1/tests/hurl/assistant.hurl
```
