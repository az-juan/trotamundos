# ADR-101: Trinidad — tres unidades modulares en lugar de N microservicios

* **Estado:** Propuesto (rama `main-trinity`)
* **Fecha:** 2026-10-28
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** I (re-elección del estilo macro / referencia)

## Contexto y Planteamiento del Problema

La rama `main` documenta Trotamundos como **microservicios hexagonales**: 7 deployables (Gateway, Web
BFF, Mobile BFF, Itinerary, Email Ingest, Integration, Analytics), cada uno con su propio `go.mod`,
sin base de datos compartida y colaborando por NATS + REST (ADR-001, ADR-002). Ese diseño satisface los
drivers de escalabilidad y aislamiento, pero **adelanta complejidad operativa** frente al contexto real
del proyecto: **equipo pequeño, poco tiempo y presupuesto limitado** (`../drivers/constraints.md`), y la
regla de sobriedad operativa de la Spec (Regla 3: *Just Enough Architecture* + *YAGNI*, Bass ch.20).

El problema es elegir un **estilo macro (C&C de referencia, Bass ch.20; Newman ch.1)** que conserve los
beneficios de la separación por responsabilidades y perfiles de carga opuestos, **sin** pagar el
overhead de 7 servicios independientes (7 pipelines, 7 artefactos, descubrimiento, mesh, DB por
servicio, observabilidad multiplicada).

## Drivers de Decisión

* **CON-5 / Iteración I** — elección de arquitectura de referencia y estilos arquitectónicos.
* **CON-3** — documentar con vistas + ADRs (MADR).
* **Constraints** — equipo pequeño, poco tiempo, presupuesto limitado (Regla 3 de la Spec).
* **QA-1/QA-2 (escalabilidad)** y **QA-3 (picos masivos)** — requieren separar los perfiles de carga opuestos (batch CPU vs. per-request).
* **QA-11 (modifiability)** — extensibilidad para nuevos medios de transporte sin fricción estructural.
* **QA-6/QA-7 (rendimiento)** — web ≤ 800 ms, FCP móvil < 1.4 s.

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *Microservicios finos (status quo de `main`)* — 7 deployables, `go.mod` por servicio,
  sin DB compartida, NATS para toda colaboración async.
* **Opción 2:** *Monolito modular único (1 deployable)* — todo el sistema en un solo proceso y un solo
  contenedor, dividido internamente en módulos hexagonales; sin comunicación entre procesos.
* **Opción 3:** *Monolito modular partido en 3 unidades (Trinidad)* — tres módulos desplegables
  (Pater/Filius/Spiritus), cada uno agrupando servicios afines; comunicación en proceso dentro de la
  unidad y por red/eventos solo entre unidades. **(propuesta)**
* **Opción 4:** *SOA / servicios flojamente acoplados con ESB ligero* — servicios con un bus de
  integración central para orquestación y transformación.

## Evaluación Comparativa de Alternativas

### Opción 1: Microservicios finos (status quo de `main`)
* **Ventajas:** máximo escalado independiente; aislamiento de fallos fuerte; cada servicio evoluciona y
  se despliega por separado; maduro para equipos grandes.
* **Desventajas:** 7 pipelines y artefactos; overhead de red y de trazabilidad distribuida; sinergia
  baja para un equipo de 3–5 devs; la coordinación de deploys se vuelve cuello de botella (Newman ch.15).
* **Impacto en Restricciones:** **contradice** equipo pequeño / poco tiempo / presupuesto (Regla 3). El
  costo operativo (7 servicios) es desproporcionado para la Fase 0.

### Opción 2: Monolito modular único (1 deployable)
* **Ventajas:** deploy, test y debug triviales; cero overhead de red; un solo artefacto; ideal para
  arrancar (Newman ch.1: *start monolithic*).
* **Desventajas:** no permite **escalado independiente** de los perfiles de carga opuestos (el escáner
  de correo bursty de CPU y el panel per-request de baja latencia competirían por recursos); un fallo
  en el escáner puede tumbar el panel; no explota QA-3/QA-1 de forma limpia.
* **Impacto en Restricciones:** compatible con el equipo pequeño, pero **insuficiente** para los drivers
  de escalabilidad y mitigación de picos (QA-1, QA-3).

### Opción 3: Monolito modular partido en 3 unidades (Trinidad) — propuesta
* **Ventajas:** conserva el **escalado y aislamiento por perfil de carga** (Pater latency, Filius OLTP,
  Spiritus batch/OLAP) con **solo 3 deployables**; comunicación en proceso dentro de la unidad (sin
  red para lo que no se necesita separar); vía de extracción preservada (módulos → servicios si el
  perfil diverge, ADR-102); un solo despliegue `docker compose`.
* **Desventajas:** introduce **3 límites de proceso** (monolito distribuido) que exigen fronteras bien
  puestas; si una frontera es incorrecta, se obtiene el peor de ambos mundos (Newman ch.1).
* **Impacto en Restricciones:** **alineado** — 3 deployables (no 7), sin mesh/K8s, un Droplet, Docker
  Compose. Respeta equipo pequeño, tiempo y presupuesto, y a la vez cubre QA-1/QA-3.

### Opción 4: SOA con ESB ligero
* **Ventajas:** reutilización de servicios vía bus; orquestación centralizada.
* **Desventajas:** el ESB tiende a convertirse en «cerebro» frágil y en punto central de acoplamiento
  (anti-pattern); complejidad de integración mayor que la que se busca evitar; poco apto para equipos
  pequeños (Newman ch.1 critica la sobrecarga de integración central).
* **Impacto en Restricciones:** **peor** que la Opción 1 en complejidad; descartada por sobre-ingeniería.

## Decisión Adoptada y Justificación

Se elige la **Opción 3 (Trinidad)**. Es el punto óptimo del espectro de monolitos de Newman (ch.1):
parte de un monolito **ya particionado por la realidad de los perfiles de carga** (el mismo argumento
que justifica ADR-001: *escalado independiente de perfiles opuestos*), pero **colapsa** los 7 servicios
finos en **3 unidades** porque, en Fase 0, los servicios que caen en cada unidad comparten perfil
(per-request/OLTP en Filius; batch/async en Spiritus; presentación en Pater). Se obtiene el 80% del
valor de los microservicios (escalado e aislamiento por carga) con una fracción del overhead, y se
conserva la **vía de extracción** (ADR-102) para cuando un módulo interno merezca convertirse en
servicio. La Opción 2 se descarta por no cubrir QA-1/QA-3; la Opción 1 se descarta por violar Regla 3;
la Opción 4 por sobre-ingeniería.

## Consecuencias

* **Impacto Positivo:** 7 → 3 deployables (4 pipelines y artefactos menos); un solo `docker compose`;
  comunicación en proceso dentro de cada unidad; stack y decisiones técnicas **sin cambios** (Go,
  hexagonal, PostgreSQL/Redis/ClickHouse, NATS, Chi, templ/htmx, JWT RS256, GitHub Actions, DO).
* **Trade-offs y Riesgos Aceptados:** riesgo de *monolito distribuido* si las fronteras fallan
  (mitigado: fronteras por perfil de carga + todo en proceso dentro de la unidad); 3 saltos de red
  entre unidades exigen timeouts y degradación (ADR-103).
* **Deuda Arquitectónica / Acciones Futuras:** mantener los módulos internos hexagonales para que la
  extracción sea un `git mv` + nuevo `go.mod` (ADR-102); revisar a Fase 1 si el escalado de una unidad
  demanda réplicas o separación de módulo a servicio.
