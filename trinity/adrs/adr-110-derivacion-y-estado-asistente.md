# ADR-110: Derivación a la agencia preferida, estado del caso y evento de asistencia

* **Estado:** Aceptado
* **Fecha:** 2026-11-22
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** IV (extensión de mejoras, CON-5; despliegue previsto en **Fase 1**)

## Contexto y Planteamiento del Problema

ADR-109 ubica el Asistente Virtual en Pater. Queda decidir **cómo se representa y se entrega un caso
complejo** a la agencia de viajes preferida del usuario, cómo se consulta su estado y qué se registra
para medir QA-17, sin violar FR-6 (deep-link a terceros), ADR-103 (NATS solo en `Filius↔Spiritus`),
ADR-104/108 (propiedad y RBAC por esquema) ni QA-9 (PII).

La preferencia ya existe: `filius.users.preferences` (`JSONB`, ADR-106) admite un campo
`preferred_agency_id` sin nueva tabla ni migración estructural.

## Drivers de Decisión

* **FR-11 (nuevo)** — derivar casos complejos a la agencia preferida con el contexto mínimo necesario.
* **FR-6** — redirección a apps de terceros para atención al cliente (reutilizar, no duplicar).
* **QA-17 (nuevo)** — medir la tasa de resolución de primer nivel desde eventos.
* **QA-12** — trazabilidad del caso con `correlation-id`.
* **ADR-103 / ADR-104 / ADR-108** — una sola arista NATS; propiedad estricta del esquema.
* **QA-9 / Datensparsamkeit** — no persistir la conversación ni exponer PII al proveedor de LLM.

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *Sesión y casos en el esquema `pater`* — Pater persiste `pater.chat_sessions` y `pater.assistance_cases`.
* **Opción 2:** *Caso en Filius + evento en la arista existente* — `filius.assistance_cases` (dueño: Itinerary), creado por REST desde Pater; Filius publica `assistance.handoff_requested`; sesión efímera en Redis. **(propuesta)**
* **Opción 3:** *Plataforma externa de tickets B2B* — derivar a un sistema de tickets de la agencia con API propia.
* **Opción 4:** *Solo deep-link sin persistencia* — generar la URL y no registrar estado ni métricas.

## Evaluación Comparativa de Alternativas

### Opción 1: Sesión y casos en `pater`
* **Ventajas:** Pater es dueño de sus datos; consulta local rápida.
* **Desventajas:** el caso es un hecho de dominio (reserva/viaje/usuario) y Pater **no tiene dominio** (ADR-102); Spiritus no podría medir QA-17 sin una nueva arista de lectura; duplica la noción de "caso" frente a Filius.
* **Impacto en Restricciones:** rompe la separación de responsabilidades; descartada.

### Opción 2: Caso en Filius + evento en la arista existente — propuesta
* **Ventajas:** el agregado `AssistanceCase` vive donde viven `Trip`/`Booking`/`users` (Filius, ADR-104); Pater solo lo crea/consulta por REST (arista permitida); Filius publica `assistance.handoff_requested` y `assistance.resolved` por **NATS en la única arista** `Filius↔Spiritus`, donde Analytics ya consume y calcula QA-17; la sesión de chat queda en Redis con TTL (sin PII persistida); el contexto para la agencia viaja en un **token de un solo uso** en Redis (TTL 15 min) y el enlace usa el patrón de deep-link de FR-6.
* **Desventajas:** dos escrituras locales (case + token en Redis) con compensación explícita si falla la segunda; consistencia eventual en la métrica (aceptada, AP — Newman ch.12).
* **Impacto en Restricciones:** óptima — cero deployables, cero aristas nuevas, un solo esquema escritor.

### Opción 3: Plataforma externa de tickets B2B
* **Ventajas:** workflow de agencia completo (SLA, asignación, auditoría).
* **Desventajas:** nuevo proveedor, coste, contrato y PII fuera del sistema; contradice constraints y QA-9; acoplamiento a la agencia concreta.
* **Impacto en Restricciones:** incompatible con Fase 1 bajo las restricciones actuales. Descartada.

### Opción 4: Solo deep-link sin persistencia
* **Ventajas:** mínimo código.
* **Desventajas:** imposible responder "¿en qué está mi derivación?" (consultas de estado), imposible medir QA-17 ni auditar (QA-12).
* **Impacto en Restricciones:** no satisface FR-10/FR-11/QA-17. Descartada.

## Decisión Adoptada y Justificación

Se elige la **Opción 2**. El flujo es una **saga con transacciones ACID locales** (Newman ch.6), sin 2PC:

1. Pater (`assistant/`) determina que la intención es compleja y llama `POST /internal/assistance/cases`
   en Filius (JWT interno, `X-Correlation-ID`, timeout) con `user_id`, `booking_id?`, `motivo` y `severidad`.
2. Filius crea `filius.assistance_cases` en estado `open` (tx local) y publica
   `assistance.handoff_requested` con retry en memoria (ADR-108).
3. Filius/Agencia: el usuario recibe un **deep-link** (FR-6) hacia la agencia configurada en
   `filius.users.preferences.preferred_agency_id`, con un `context_token` **opaco** (ULID) en Redis
   (TTL 15 min, un solo uso) que contiene el contexto mínimo del caso — nunca la transcripción.
4. La agencia consulta el contexto con ese token; el caso pasa a `handed_off`. Un webhook/envío del
   proveedor o una actualización del usuario lo cierra (`resolved` / `closed`) y Filius publica
   `assistance.resolved` → Spiritus actualiza la métrica de QA-17.

**Estados del caso:** `open → handed_off → resolved → closed` (con `expired` desde `open`/`handed_off`
si el token vence). El token y la sesión de chat se almacenan en **Redis**, que ADR-104 ya define como
caché compartido sin dueño de dominio.

**Eventos:** nuevos subjects `assistance.handoff_requested` y `assistance.resolved` en el stream
`ASSISTANCE` (`assistance.>`, retención 30 d). Es el **mismo** conector NATS de la única arista permitida;
no se añade consumidor en Pater.

## Consecuencias

* **Impacto Positivo:** derivación trazable y auditable; consulta de estado del caso; medición de QA-17 desde Spiritus sin tocar el OLTP de Filius; PII minimizada; FR-6 reutilizado.
* **Trade-offs y Riesgos Aceptados:** el token caduca en 15 min (el usuario puede solicitar uno nuevo); la métrica QA-17 es eventual; la agencia concreta debe soportar el parámetro de contexto del deep-link (si no, se degrada al deep-link genérico de FR-6).
* **Deuda Arquitectónica / Acciones Futuras:** si el volumen de casos exige SLA propio, evaluar extracción del módulo (regla 2-de-3, ADR-102) o plataforma B2B dedicada con un ADR de revisión.
