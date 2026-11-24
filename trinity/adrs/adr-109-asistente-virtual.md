# ADR-109: Asistente Virtual como módulo de Pater con clasificador de intenciones intercambiable

* **Estado:** Aceptado (Implementado en inicio parcial de Fase 1 con clasificador determinista, ver [ADR-113](./adr-113-inicio-parcial-fase1-asistente-determinista.md))
* **Fecha:** 2026-11-22
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** IV (extensión de mejoras, CON-5; activo en **Fase 1 parcial**)

## Contexto y Planteamiento del Problema

Se solicita incorporar un **Asistente Virtual (chatbot)** como primer nivel de soporte: gestionar
reprogramaciones, consultas de estado e incidencias menores, y delegar a la **agencia de viajes
preferida del usuario** solo los casos complejos.

**Auditoría de consistencia previa (SPEC-001, Regla 2 / directiva 1 — Paso 0):**

| ID | Conflicto detectado con la línea base | Resolución adoptada |
|----|----------------------------------------|---------------------|
| C1 | `drivers/design-purpose.md` §5 excluye **«chat»** del scope de **Fase 0**. | La capacidad se **prove en Fase 1**; Fase 0 queda intacta. Se aclara el driver en vez de revocarlo silenciosamente. |
| C2 | `drivers/requerimientos.md` **FR-6** ya redirige al usuario a apps de terceros para *atención al cliente*. | La derivación **reutiliza FR-6** (deep-link); no se crea un canal B2B nuevo. |
| C3 | **ADR-103** restringe NATS a la arista `Filius↔Spiritus`; Pater no publica ni consume eventos. | El asistente vive en **Pater** y solo usa la arista REST `Pater↔Filius`; Filius publica el evento de derivación. **No hay cuarta arista.** |
| C4 | `drivers/constraints.md`: ~$80/mes, sin plataformas gestionadas en Fase 0 (Regla 3, YAGNI). | Proveedor de LLM **solo en Fase 1**, detrás de un puerto, con redacción de PII, presupuesto vigilado y fallback determinista. |
| C5 | **ADR-108 / ADR-104**: RBAC por esquema; nadie escribe en el esquema ajeno. | El estado efímero va en **Redis** (sin dueño de dominio); el caso de asistencia lo posee **Filius**. |
| C6 | **QA-6 / QA-7** (web ≤ 800 ms, FCP < 1.4 s) y **QA-9** (PII). | Respuesta **streaming** por htmx/SSE sin bloquear el render; la conversación **no se persiste** y el LLM recibe solo contexto redactado. |

Ningún cambio contradice un ADR vigente: se trata de una **ampliación de la Iteración IV** con destino Fase 1.

## Drivers de Decisión

* **FR-10 / FR-11 (nuevos)** — asistencia de primer nivel y derivación a la agencia preferida.
* **QA-16 (nuevo)** — primera respuesta conversacional p95 < 2 s sin degradar QA-6/QA-7.
* **QA-17 (nuevo)** — ≥ 70% de intenciones resueltas sin intervención humana (medible desde eventos).
* **QA-9 (seguridad)** — mínima PII al proveedor de LLM; consentimiento explícito.
* **QA-3 / QA-8** — degradación elegante si el proveedor externo falla; sin afectar al núcleo transaccional.
* **QA-11 / ADR-102** — extensibilidad sin cambiar la topología de 3 unidades.
* **Constraints / Regla 3** — equipo pequeño, presupuesto acotado, *Just Enough Architecture*.

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *Plataforma SaaS de chatbot* (Dialogflow / Lex / Bot Framework) integrada desde Pater.
* **Opción 2:** *Servicio de Conversación independiente* (4.º deployable con su propio `go.mod`, store y broker).
* **Opción 3:** *Módulo `assistant/` dentro de Pater* con puerto `IntentClassifier`, adaptador LLM tras ACL y fallback determinista. **(propuesta)**
* **Opción 4:** *Asistente puramente determinista* (árbol de reglas/keywords, sin modelo de lenguaje).

## Evaluación Comparativa de Alternativas

### Opción 1: Plataforma SaaS de chatbot
* **Ventajas:** time-to-market corto; NLU administrado; UI de entrenamiento incluida.
* **Desventajas:** coste por sesión impredecible frente a ~$80/mes; envía conversación (con PII de viaje) a un tercero; acoplamiento a un vendor y a su modelo de datos; contradice la Datensparsamkeit de `ops/operaciones.md`.
* **Impacto en Restricciones:** **contradice** Regla 3 y QA-9. Descartada.

### Opción 2: Servicio de Conversación independiente
* **Ventajas:** aislamiento de fallos y escalado propio; despliegue separado del Rostro.
* **Desventajas:** rompe la Trinidad (4 deployables), exige una cuarta arista o mongo de NATS que ADR-103 prohíbe, añade pipeline, secretos y observabilidad; la regla 2-de-3 (ADR-102) no se cumple en Fase 1.
* **Impacto en Restricciones:** sobre-ingeniería; **contradice** ADR-101/103. Descartada.

### Opción 3: Módulo `assistant/` en Pater — propuesta
* **Ventajas:** reutiliza el borde existente (JWT, rate limit, SSE, correlation-id); la consulta de estado usa la arista REST ya definida `Pater↔Filius`; el puerto `IntentClassifier` permite sustituir el proveedor o usar reglas sin tocar casos de uso; el fallback determinista cumple QA-8/QA-3 cuando el LLM falla; cero deployables nuevos.
* **Desventajas:** comparte perfil de latencia con el Rostro (mitigado: streaming, timeouts y no bloquear el render inicial); requiere disciplina para no persistir conversación.
* **Impacto en Restricciones:** **alineado** — 3 unidades, sin nueva infraestructura en Fase 0, coste gobernado en Fase 1.

### Opción 4: Asistente puramente determinista
* **Ventajas:** coste $0, sin PII externa, máxima testeabilidad.
* **Desventajas:** cobertura de lenguaje natural baja; difícil alcanzar QA-17 (≥ 70% de resolución) con frases libres.
* **Impacto en Restricciones:** compatible, pero **insuficiente** como solución principal; se conserva como *fallback* de la Opción 3.

## Decisión Adoptada y Justificación

Se elige la **Opción 3**, con la **Opción 4 como mecanismo de degradación**. El asistente es un módulo
hexagonal más dentro de `units/pater` (`assistant/`), junto a `gateway/`, `web-bff/` y `mobile-bff/`.
Cumple el principio *one BFF per experience* (ADR-013): `web-bff` sirve el chat por htmx/SSE y
`mobile-bff` sirve el mismo contrato en JSON; el Gateway aplica JWT RS256, rate limit y trazabilidad.

Puertos y dependencias (ADR-002, flujo hacia el centro):

| Dirección | Contrato | Implementación |
|-----------|----------|----------------|
| Entrada | `ChatPort` | `POST /assistant/messages`, `GET /assistant/stream` (SSE) |
| Salida | `FiliusClient` (REST) | consultas de estado y creación del caso (arista `Pater↔Filius`) |
| Salida | `IntentClassifier` | adaptador LLM con **ACL + redacción de PII** (Fase 1) / adaptador `RuleBasedClassifier` (fallback) |
| Salida | `SessionStore` | Redis con TTL corto; **sin transcripción persistida** |

**Reglas de oro:** el asistente **no escribe en proveedores** — la reprogramación se instrumenta como
un *deep-link* (FR-6) y su estado se confirma cuando llega el evento del proveedor; los casos complejos
se derivan según [ADR-110](./adr-110-derivacion-y-estado-asistente.md). Si el clasificador externo
falla, aplica circuit breaker (`shared/resilient`) y se responde con el clasificador determinista
más una opción de derivación: la conversación nunca queda sin respuesta (QA-8/QA-3).

## Consecuencias

* **Impacto Positivo:** nueva capacidad sin tocar la topología ni los contratos de las 3 aristas; latencia del Rostro preservada; proveedor de LLM intercambiable; trazabilidad extremo a extremo con `correlation-id`.
* **Trade-offs y Riesgos Aceptados:** el chat comparte proceso con Pater (un pico de conversaciones compite con el render); mitigado por rate limit, streaming y timeouts. Calidad de respuestas dependiente del proveedor externo; mitigado por fallback y medición de QA-17.
* **Deuda Arquitectónica / Acciones Futuras:** `DT-06` (coste/PII del LLM externo, ver catálogo) se retira o reduce si el volumen justifica un adaptador propio; si el módulo cumple 2-de-3 (ADR-102) se extrae a servicio con un nuevo ADR de revisión.
