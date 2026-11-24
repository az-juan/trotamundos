# ADR-113: Inicio parcial de Fase 1 — MVP determinista del Asistente Virtual sin LLM externo

* **Estado:** Aceptado
* **Fecha:** 2026-11-23
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** IV extendida (transición incremental a **Fase 1 parcial** por Disparador 5)

## Contexto y Planteamiento del Problema

Cerrado y verificado el prototipo de **Fase 0** (CON-4, ADR-111/112), se acuerda dar inicio a la
**Fase 1** activando **únicamente el Asistente Virtual (chatbot)** como primer nivel de soporte
(`FR-10`, `FR-11`, `QA-16`, `QA-17`), **sin** adelantar los demás elementos previstos para Fase 1
(2.º Droplet con ClickHouse + Metabase, Mobile BFF JSON, JWT RS256/OIDC ni extracción del módulo
Notification) y priorizando un **MVP simple y determinista en código pero rigurosamente documentado**.

**Auditoría de consistencia previa (SPEC-001, Regla 2 / Directiva 1 — Paso 0):**

| ID | Elemento de la línea base | Evaluación de consistencia | Resolución adoptada |
|----|---------------------------|----------------------------|---------------------|
| C1 | `design-purpose.md` §5 y `ADR-109` (C1) excluyen el chat de **Fase 0** y lo asignan a **Fase 1**. | Se respeta la frontera: no se altera Fase 0 rétroactivamente; se **inicia parcialmente la Fase 1** activada por el Disparador 5 de `registers/transiciones.md`. | Se crea la carpeta `fase_1/` (sin duplicar código ni `docker-compose.yml`) para reflejar el inicio parcial de Fase 1 exclusivamente con el chatbot. |
| C2 | `registers/transiciones.md` agrupa en Fase 1 ClickHouse, Redis/NATS managed, Notification y Asistente Virtual. | El criterio de entrada a Fase 1 en `transiciones.md` es disyuntivo (*«volumen de eventos, notificación en vivo **o** disponibilidad del canal de soporte conversacional»*). | En virtud de la **Regla 1** (*No adelantarse*) y **Regla 3** (*YAGNI*), se activa solo el Asistente Virtual; ClickHouse (`DT-01`) y Mobile BFF (`DT-08`) quedan pendientes de sus propios disparadores. |
| C3 | `ADR-109` define el puerto `IntentClassifier` con adaptador LLM externo + fallback determinista (`RuleBasedClassifier`), registrando la deuda `DT-06` (coste/PII del LLM). | Un LLM externo añade complejidad, gestión de secretos y gasto variable innecesarios para validar las fronteras arquitectónicas del MVP de Fase 1. | El MVP de Fase 1 instancia `IntentClassifier` **exclusivamente con `RuleBasedClassifier`** (determinista, coste $0, cero PII externa, latencia < 5 ms), dejando el puerto listo para conectar un LLM futuro sin tocar casos de uso. |
| C4 | `ADR-111` adoptó SSR con `templ` + `HTMX` + `sqlc` y cookie firmada para la interfaz web en Pater, mientras que `ADR-110` exige que el caso complejo se coordine con Filius (`POST /internal/assistance/cases`), Redis (`context_token` TTL 15 min) y NATS (`assistance.*` → Spiritus). | Ambos diseños son ortogonales y compatibles: la UI del viajero en Pater puede servirse por **HTMX** (`/assistant`, `POST /assistant/messages`) mientras Pater usa Redis para la sesión efímera e invoca a Filius por REST para la saga de derivación y publicación en NATS. | Se combinan ambos: interfaz 100 % HTMX en Pater (sin JS propio) + sesión/token efímeros en Redis + arista REST `Pater→Filius` + eventos `assistance.>` en NATS JetStream hacia Spiritus. |

## Drivers de Decisión

* **FR-10 / FR-11** — soporte interactivo de primer nivel (estado, reprogramación guiada, incidencias menores) y derivación trazable de casos complejos a la agencia preferida (`preferred_agency_id`) reutilizando `FR-6`.
* **QA-16** — respuesta conversacional p95 < 2 s sin degradar el render web (`QA-6` ≤ 800 ms).
* **QA-17** — medición de la tasa de resolución autónoma de Nivel 1 desde eventos `assistance.*` en Spiritus.
* **QA-9** — redacción de PII antes de clasificar y no persistencia de la transcripción del chat en base de datos relacional.
* **Regla 1, Regla 2 y Regla 3 de `SPEC_ARCHITECTURE.md`** — no adelantar ClickHouse/Mobile BFF; sobriedad operativa; *Just Enough Architecture*.

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *Bloqueo hasta Fase 1 completa* — no implementar el chatbot hasta desplegar el 2.º Droplet con ClickHouse, Metabase, Mobile BFF JSON, JWT RS256 y un proveedor LLM comercial.
* **Opción 2:** *Chatbot aislado en Pater solo con SQL* — implementar una vista HTMX en Pater que escriba en PostgreSQL sin usar Redis, sin invocar a Filius por REST y sin publicar en NATS JetStream.
* **Opción 3:** *Inicio parcial de Fase 1 con MVP determinista sobre la topología completa de ADR-109/110 e interfaz HTMX de ADR-111* — vista HTMX en Pater + `RuleBasedClassifier` detrás de `IntentClassifier` + sesión/token en Redis + REST `Pater→Filius` + eventos `assistance.>` en NATS hacia Spiritus. **(propuesta)**

## Evaluación Comparativa de Alternativas

### Opción 1: Bloqueo hasta Fase 1 completa (ClickHouse + Mobile BFF + LLM externo)
* **Ventajas:** entrega todos los hitos de Fase 1 en un solo corte.
* **Desventajas:** multiplica el costo de infraestructura ($96/mes adicionales del Droplet 2 + tokens LLM) y el esfuerzo de desarrollo por 5× cuando el único requerimiento solicitado es el canal de asistencia web.
* **Impacto en Restricciones:** **viola** la Regla 1 (*No adelantarse*) y la Regla 3 (*YAGNI*) de la SPEC. Descartada.

### Opción 2: Chatbot aislado en Pater solo con SQL (sin Redis, sin Filius REST, sin NATS)
* **Ventajas:** mínima cantidad de líneas de código en una sola unidad (`units/pater`).
* **Desventajas:** no valida la arquitectura distribuida de [ADR-109](./adr-109-asistente-virtual.md) y [ADR-110](./adr-110-derivacion-y-estado-asistente.md) (no prueba la expiración de `context_token` de un solo uso en Redis ni el consumo asíncrono de `assistance.>` en NATS por Spiritus).
* **Impacto en Restricciones:** incumple la topología acordada en ADR-109/110. Descartada.

### Opción 3: Inicio parcial de Fase 1 con MVP determinista sobre topología ADR-109/110 + UI HTMX (ADR-111) — propuesta
* **Ventajas:**
  1. Valida de extremo a extremo la arquitectura de las 3 unidades para el Asistente Virtual: Pater (`internal/assistant` + vistas `templ`/HTMX + sesión efímera en Redis TTL 30 min) → Filius (`POST /internal/assistance/cases` + `filius.assistance_cases` + `context_token` de un solo uso en Redis TTL 15 min) → Spiritus (suscripción NATS `assistance.>` + cálculo de `QA-17`).
  2. Mantiene el código del clasificador simple, predecible y auditable (`RuleBasedClassifier` detrás de `IntentClassifier`), con coste $0, latencia sub-milisegundo (`QA-16`) y cero fugas de PII a terceros (`QA-9`, neutralizando el interés de la deuda `DT-06`).
  3. Evita duplicar código o infraestructura: reutiliza `fase_0/deploy/docker-compose.yml` y organiza las pruebas y guía del nuevo hito en `fase_1/`.
* **Desventajas:** comprensión de lenguaje natural acotada al vocabulario de reglas del dominio de viajes hasta que se conecte un adaptador LLM en una iteración posterior.
* **Impacto en Restricciones:** **óptimo** — respeta presupuesto, equipo pequeño y coherencia documental (Reglas 1, 2, 3 y 4 de la SPEC).

## Decisión Adoptada y Justificación

Se elige la **Opción 3**. Se declara formalmente el **inicio parcial de la Fase 1**, acotado de manera
exclusiva al **MVP determinista del Asistente Virtual**:

1. **Interfaz y Clasificación en Pater (`units/pater/internal/assistant/`):**
   - Se incorpora la ruta `/assistant` en la navegación global y el endpoint `POST /assistant/messages` renderizado con `templ` y `HTMX` (sin JavaScript propio, conforme a ADR-111).
   - Todo mensaje pasa por `RedactPII` (`QA-9`) antes de evaluarse en `IntentClassifier` (`RuleBasedClassifier`), que discrimina las 5 intenciones del contrato (`status_query`, `reschedule_guidance`, `minor_issue`, `complex_handoff`, `out_of_scope`).
   - Los turnos de la conversación se almacenan únicamente en **Redis** (`assistant:session:{id}`, TTL 30 min); jamás se persiste la transcripción en PostgreSQL.
2. **Derivación y Token de Contexto en Filius (`units/filius/internal/api/assistance.go`):**
   - Ante un `complex_handoff`, Pater persiste el caso en `filius.assistance_cases` e invoca `POST /internal/assistance/cases` en Filius con `Idempotency-Key` y `X-Correlation-ID`.
   - Filius genera un `context_token` opaco en **Redis** (TTL 15 min, consumible una sola vez con `GETDEL`), construye el *deep-link* (`FR-6`) hacia la agencia preferida (`preferred_agency_id`) y publica `assistance.handoff_requested` en NATS JetStream.
3. **Analítica `QA-17` en Spiritus (`units/spiritus/cmd/spiritus/main.go`):**
   - Spiritus añade la suscripción durable `spiritus-assistance` al subject `assistance.>` en NATS JetStream, procesando `assistance.handoff_requested` y `assistance.resolved` sin impactar el OLTP (`QA-15`).
4. **Organización del repositorio (`fase_1/`):**
   - Se añade el directorio `fase_1/` únicamente con la documentación del incremento (`fase_1/README.md`) y sus pruebas E2E (`fase_1/tests/hurl/assistant.hurl`), sin duplicar `fase_0/deploy/`.

## Consecuencias

* **Impacto Positivo:** el Asistente Virtual deja de ser solo diseño en papel y pasa a ser una capacidad ejecutable y verificada por tests unitarios y E2E (`./scripts/test.sh` y `./scripts/hurl.sh`), cumpliendo `FR-10`, `FR-11`, `QA-16` y `QA-17` con cero costo adicional de infraestructura.
* **Trade-offs y Riesgos Aceptados:** la Fase 1 queda en estado **En curso (parcial: Asistente Virtual activo)**; el clasificador es determinista por palabras clave del dominio.
* **Deuda Arquitectónica / Acciones Futuras:** `DT-06` (riesgo de coste/PII de LLM externo) permanece con interés **$0** mientras opere `RuleBasedClassifier`; `DT-01` (ClickHouse) y `DT-08` (Mobile BFF + JWT) permanecen diferidos hasta que se activen sus disparadores específicos en `registers/transiciones.md`.
