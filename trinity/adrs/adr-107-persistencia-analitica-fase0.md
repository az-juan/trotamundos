# ADR-107: Estrategia de Persistencia Analítica y Read-Model (`Spiritus`) en Fase 0

* **Estado:** Aceptado
* **Fecha:** 2026-11-21
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** IV (refinamiento de persistencia por fase y preparación del prototipo)

## Contexto y Planteamiento del Problema

Las restricciones del proyecto ([constraints.md](../drivers/constraints.md)) fijan un presupuesto de **~$80/mes para la Fase 0** (Droplet 1 de $48/mes + PostgreSQL Managed de $15–30/mes), posponiendo el **2.º Droplet dedicado a ClickHouse + Metabase ($96/mes)** para la **Fase 1** (ADR-104, ADR-105). Sin embargo, el alcance de la Fase 0 ([design-purpose.md](../drivers/design-purpose.md)) y el contrato OpenAPI ([openapi.yaml](../../fase_0/docs/api/openapi.yaml)) incluyen el endpoint `GET /insights` (`FR-7`, `QA-15`) servido por el módulo `analytics` de **Spiritus** hacia **Pater**.

Existía un vacío de diseño: si en Fase 0 no se despliega el servidor ClickHouse, ¿qué adaptador de persistencia respalda al módulo `analytics` de **Spiritus** para almacenar el histórico de eventos de dominio y servir el read-model de insights sin violar el presupuesto ni degradar las tablas OLTP de **Filius**?

## Drivers de Decisión

* **Constraints** — presupuesto estricto de ~$80/mes en Fase 0 (sin 2.º Droplet en Fase 0).
* **QA-15 (Escalabilidad analítica)** — aislar las lecturas analíticas de la operación transaccional de reservas (`filius`).
* **FR-7 / FR-9 (Recopilación analítica e insights)** — disponibilidad de `GET /insights` desde Fase 0.
* **ADR-002 (Arquitectura Hexagonal por módulo)** — puertos de salida desacoplados de la tecnología de almacenamiento.
* **ADR-104 (Segregación por esquema)** — esquema `spiritus` disponible en la instancia PostgreSQL compartida.

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *Polimorfismo Hexagonal por Fase — Adaptador PostgreSQL (`spiritus.analytics_events` + `spiritus.insight_aggregates`) en Fase 0 → Adaptador ClickHouse en Fase 1*. **(propuesta)**
* **Opción 2:** *Contenedor ClickHouse Single-Node embebido en el Droplet 1 (`docker-compose.yml`) con límite estricto de RAM desde Fase 0*.
* **Opción 3:** *Motor OLAP Columnar Embebido en Proceso (`DuckDB`) en Fase 0 → ClickHouse en Fase 1*.
* **Opción 4:** *Pre-agregación en Tiempo Real sobre Redis (`Hashes` / `Sorted Sets`) con Snapshot en PostgreSQL*.

## Evaluación Comparativa de Alternativas

### Opción 1: Polimorfismo Hexagonal por Fase (Adaptador PostgreSQL en esquema `spiritus` en Fase 0 → ClickHouse en Fase 1) — propuesta
* **Ventajas:** Costo de infraestructura **$0 adicional en Fase 0**; aprovecha el esquema `spiritus` ya existente en PostgreSQL sin tocar jamás las tablas del esquema `filius`; explota directamente la arquitectura hexagonal (ADR-002) mediante el puerto `AnalyticsRepository`, seleccionando `PostgresAnalyticsRepo` (`SPIRITUS_ANALYTICS_DRIVER=postgres`) en Fase 0 y `ClickHouseAnalyticsRepo` (`SPIRITUS_ANALYTICS_DRIVER=clickhouse`) en Fase 1 sin modificar `domain` ni `usecase`; conserva binarios Go estáticos (`CGO_ENABLED=0`).
* **Desventajas:** En Fase 0 las escrituras batch de `spiritus.analytics_events` comparten la instancia física de PostgreSQL Managed (aunque en tablas y esquema separados con pre-agregación en `spiritus.insight_aggregates`).
* **Impacto en Restricciones:** Cumplimiento total de presupuesto (~$80/mes), simplicidad operativa y cero contenedores adicionales en el Droplet 1.

### Opción 2: Contenedor ClickHouse Single-Node embebido en el Droplet 1 desde Fase 0
* **Ventajas:** Un único adaptador (`clickhouse-go/v2`) desde el día 1 y aislamiento físico respecto de PostgreSQL.
* **Desventajas:** Consume ~1.5 GB de los 8 GB de RAM del Droplet 1; añade un motor adicional a respaldar y monitorear en Fase 0; contradice la restricción explícita de `constraints.md` de reservar ClickHouse para la Fase 1.
* **Impacto en Restricciones:** Reduce el margen de memoria del Droplet 1 e incrementa la carga operativa inicial.

### Opción 3: Motor OLAP Columnar Embebido (`DuckDB`) en Fase 0 → ClickHouse en Fase 1
* **Ventajas:** Almacenamiento columnar vectorizado en archivo local sin levantar otro servidor ni usar PostgreSQL.
* **Desventajas:** El driver `go-duckdb` exige `CGO_ENABLED=1` y toolchain C/C++, rompiendo la compilación rápida de binarios Go estáticos puros (ADR-003) y obligando a mantener dos dialectos SQL analíticos.
* **Impacto en Restricciones:** Introduce fricción de build y CI/CD desproporcionada para un equipo pequeño.

### Opción 4: Pre-agregación en Redis con Snapshot en PostgreSQL
* **Ventajas:** Lecturas en sub-milisegundo desde Redis sin tablas de eventos.
* **Desventajas:** Solo soporta contadores fijos; pierde la capacidad de consultar eventos históricos estructurados por usuario (`user_id`), destino o ventana arbitraria, bloqueando evoluciones analíticas sobre el historial de viajes.
* **Impacto en Restricciones:** Demasiado rígido para el dominio analítico del proyecto.

## Decisión Adoptada y Justificación

Se elige la **Opción 1**. El módulo `analytics` de la unidad **Spiritus** define el puerto de salida `AnalyticsRepository` (para ingesta batch de eventos y actualización/lectura de pre-agregados) con dos implementaciones intercambiables por configuración (`SPIRITUS_ANALYTICS_DRIVER`):

1. **En Fase 0 (`postgres`):** `PostgresAnalyticsRepo` persiste los eventos consumidos de NATS JetStream en `spiritus.analytics_events` (indexados por `(event_type, occurred_at)` y `(user_id, occurred_at)`) y actualiza las proyecciones materializadas en `spiritus.insight_aggregates` (`dimension`, `key`, `period`, `metric_val`). Cuando **Pater** solicita `GET /insights`, **Spiritus** lee `spiritus.insight_aggregates` en $< 5\text{ ms}$ sin tocar el esquema `filius`.
2. **En Fase 1 (`clickhouse`):** Al incorporar el 2.º Droplet, se conmuta `SPIRITUS_ANALYTICS_DRIVER=clickhouse` para activar `ClickHouseAnalyticsRepo` sobre `events_raw` y las vistas materializadas de ClickHouse, sin alterar una sola línea de los casos de uso ni del contrato `Spiritus → Pater`.

## Consecuencias

* **Impacto Positivo:** La Fase 0 es 100% funcional y desplegable hoy mismo dentro del presupuesto de ~$80/mes; `GET /insights` funciona de extremo a extremo desde el lanzamiento; `spiritus.analytics_events` conserva todo el historial rico por `user_id` y `destination` listo para análisis avanzados.
* **Trade-offs y Riesgos Aceptados:** Durante la Fase 0, el almacenamiento analítico reside en el mismo servidor PostgreSQL Managed que el OLTP (en el esquema aislado `spiritus`). Como las escrituras son asíncronas en batch desde NATS y las lecturas de `/insights` golpean solo la tabla pre-agregada `spiritus.insight_aggregates`, el impacto sobre el SLA de 800 ms (`QA-6`) es despreciable para el volumen de arranque.
* **Deuda Arquitectónica / Acciones Futuras:** Monitorear el tamaño del esquema `spiritus` y la latencia de IOPS en PostgreSQL; cuando el volumen de eventos dispare el criterio de transición a Fase 1 ([transiciones.md](../registers/transiciones.md)), aprovisionar el Droplet 2 con ClickHouse y cambiar `SPIRITUS_ANALYTICS_DRIVER=clickhouse`.
