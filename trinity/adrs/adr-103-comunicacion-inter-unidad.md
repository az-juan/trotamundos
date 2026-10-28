# ADR-103: Comunicación inter-unidad — el triángulo de conectores

* **Estado:** Propuesto (rama `main-trinity`)
* **Fecha:** 2026-10-28
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** III (tácticas de interacción, resiliencia y desacoplamiento)

## Contexto y Planteamiento del Problema

Con tres unidades (ADR-102) hay que decidir **cómo se comunican**. En `main`, los 7 servicios se
hablaban todos por REST y NATS con múltiples conexiones. En la Trinidad, el objetivo es que **cada par
de unidades hable exactamente un idioma**, elegido por la naturaleza de la interacción (Newman ch.4:
*communication styles before technology*), y que **dentro de cada unidad no haya red** (llamada en
proceso). Las tres aristas del triángulo son distintas:

- **Pater ↔ Filius:** el cliente necesita el resultado ya → **REST síncrono**.
- **Filius ↔ Spiritus:** propagación de hechos, cada consumidor reacciona por su cuenta → **eventos
  NATS (async)**.
- **Spiritus ↔ Pater:** el Rostro muestra insights/notificaciones sin cargar el OLTP → **read-model /
  SSE**.

## Drivers de Decisión

* **QA-5 (refresco < 5 min)** — propagación async de cambios de terceros.
* **QA-6/QA-7 (rendimiento)** — REST de baja latencia Pater→Filius.
* **QA-15 (analítica sin degradar lo transaccional)** — lecturas analíticas fuera del OLTP (CQRS).
* **QA-3 (picos masivos)** — desacoplamiento async entre el batch (Spiritus) y el núcleo.
* **QA-12 (trazabilidad)** — correlation-id en todas las aristas.
* **ADR-005 (NATS)**, **ADR-006 (Chi)**, **ADR-007 (resiliencia)**, **ADR-014 (JWT RS256)** — heredados.

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *REST en todas las aristas (request-response)* — las tres unidades se llaman por HTTP
  sincrónico entre sí.
* **Opción 2:** *Event-driven en todas las aristas (NATS por todos lados)* — toda colaboración por
  eventos, incluido Pater↔Filius.
* **Opción 3:** *Conector por naturaleza de la interacción (híbrido selectivo)* — REST en Pater↔Filius,
  NATS en Filius↔Spiritus, read-model/SSE en Spiritus→Pater. **(propuesta)**
* **Opción 4:** *Shared-memory / llamadas en proceso para todo* — una sola unidad monolítica sin
  procesos separados (equivale a la Opción 2 de ADR-101; descartada por QA-1/QA-3).

## Evaluación Comparativa de Alternativas

### Opción 1: REST en todas las aristas
* **Ventajas:** simple, síncrono, fácil de depurar. **Desventajas:** acoplamiento temporal Filius↔
  Spiritus (el batch esperaría respuestas); la analítica pesada bloquearía al Rostro; sin
  desacoplamiento de picos (QA-3). **Restricciones:** choca con QA-3/QA-15.

### Opción 2: Event-driven en todas las aristas
* **Ventajas:** máximo desacoplamiento. **Desventajas:** Pater↔Filius necesita el resultado **ya**
  (render de la web, ≤ 800 ms); forzar eventos ahí añade latencia, complejidad de correlación y
  consistencia eventual donde no se pide (Newman ch.4: elegir estilo *antes* de la tecnología).
  **Restricciones:** choca con QA-6/QA-7.

### Opción 3: Conector por naturaleza de la interacción — propuesta
* **Ventajas:** cada arista usa el conector correcto: REST donde se pide latencia, eventos donde se
  propaga un hecho, read-model donde se lee analítica. **Mínimo de conectores** (sin duplicar
  idiomas); desacoplamiento de picos (QA-3) y analítica separada del OLTP (QA-15). **Desventajas:**
  tres estilos que el equipo debe dominar; requiere catálogo de eventos y contratos versionados.
  **Restricciones:** compatible (complejidad contenida: solo 2 unidades usan NATS).

### Opción 4: Shared-memory / todo en proceso
* **Ventajas:** cero red. **Desventajas:** es un solo proceso → no cubre el escalado por perfil de
  carga (QA-1/QA-3). **(equivale a monolito único; descartada en ADR-101)**

## Decisión Adoptada y Justificación

Se elige la **Opción 3**. Aplica el principio de Newman (ch.4) de elegir el **estilo de comunicación
antes de la tecnología** y por la naturaleza de la interacción:

| Arista | Conector | Por qué |
|--------|----------|---------|
| **Pater ↔ Filius** | REST (síncrono, JWT RS256, timeout) | El cliente necesita el resultado ya (QA-6/QA-7) |
| **Filius ↔ Spiritus** | NATS JetStream (eventos ricos, async) | Propagar hechos; cada consumidor reacciona (QA-3, QA-5) |
| **Spiritus → Pater** | read-model (API) + SSE | Mostrar insights/notificaciones sin tocar el OLTP (QA-15, CQRS) |

Se **restringe NATS a una sola arista**: ya no lo consumen Gateway ni BFF (como sí en `main`), lo que
reduce el catálogo de subjects y consumidores. La resiliencia de ADR-007 se aplica en la arista
Pater↔Filius (timeouts) y en las llamadas externas de Filius (breaker/retry/rate-limit). El
**correlation-id** se propaga por OTel en las tres aristas (QA-12).

## Consecuencias

* **Impacto Positivo:** tres aristas con un único conector cada una; analítica aislada del OLTP; NATS
  con menor superficie; extracción de módulo→servicio no cambia los contratos (eventos y OpenAPI ya
  versionados).
* **Trade-offs y Riesgos Aceptados:** consistencia eventual en Filius↔Spiritus (aceptada; el mundo real
  ya es inconsistente, Newman ch.12/AP); el read-model de Spiritus puede retrasarse respecto al OLTP
  (mitigado: SSE para notificaciones en tiempo casi real).
* **Deuda Arquitectónica / Acciones Futuras:** mantener idempotencia (Redis) en consumidores de eventos
  y webhooks (ADR-007); si Filius↔Spiritus exigiera consistencia fuerte, evaluar
  orquestación con compensaciones (sagas, Newman ch.6).
