# ADR-108: Aislamiento RBAC por Esquema en PostgreSQL, Resiliencia de Publicación a NATS y Reconciliación de SLA en Fase 0

* **Estado:** Aceptado
* **Fecha:** 2026-11-21
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** IV (hardening operativo, mitigación de deuda arquitectónica y preparación del prototipo)

## Contexto y Planteamiento del Problema

Al cierre de la documentación previa a la construcción del prototipo (Iteración IV, `CON-4`), el catálogo de deuda arquitectónica ([deuda-arquitectonica.md](../registers/deuda-arquitectonica.md)) registraba tres riesgos operativos abiertos para la Fase 0:

1. **Riesgo de *Content Coupling* en PostgreSQL compartido (`DT-01`, ADR-104):** Aunque los datos se dividieron en 3 esquemas (`pater`, `filius`, `spiritus`), las 3 unidades se conectaban con un único super-usuario de base de datos (`trotamundos`), posponiendo el control de acceso por rol (RBAC) hasta la Fase 2.
2. **Riesgo de pérdida de eventos en NATS JetStream de 1 nodo (`DT-05`, ADR-103):** El servicio `nats` en `docker-compose.yml` no tenía montado un volumen persistente para el *file store* de JetStream (`-sd /data`), y la publicación de eventos tras un `COMMIT` en PostgreSQL carecía de una política explícita de reintento ante micro-cortes del broker.
3. **Tensión entre el driver de disponibilidad `QA-8` y la topología de 1 Droplet (`DT-03`, ADR-105):** El escenario `QA-8` ([atributos.md](../drivers/atributos.md)) establece un objetivo de negocio ideal de $\le 5\text{ minutos}$ de caída al mes (**99.99%**), mientras que la restricción de presupuesto de Fase 0 (~$80/mes, 1 solo Droplet sin auto-healing multi-nodo) solo permite comprometer razonablemente un SLO operativo de **99.9%** ($\approx 43.2\text{ min/mes}$).

## Drivers de Decisión

* **QA-8 (Disponibilidad 24/7)** — minimizar el tiempo de caída y evitar pérdida de estado en reinicios.
* **QA-5 (Refresco < 5 min)** — entrega confiable de eventos en la arista `Filius ↔ Spiritus`.
* **QA-9 (Seguridad)** — principio de mínimo privilegio en el acceso a datos desde Fase 0.
* **QA-11 (Modificabilidad)** — garantía impuesta por infraestructura de que ninguna unidad accede al esquema de otra.
* **Constraints** — presupuesto de ~$80/mes en Fase 0 y equipo pequeño de 3–5 desarrolladores (Regla 3 de la SPEC).

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *RBAC Estricto por Esquema en PG + Patrón Transactional Outbox en SQL + Volumen Persistente JetStream*.
* **Opción 2:** *RBAC Estricto por Esquema en PG + Publicación Síncrona con Reintento en Memoria (`Exponential Backoff`) + Volumen Persistente JetStream (Sin Tabla Outbox)*. **(propuesta)**
* **Opción 3:** *RBAC Estricto por Esquema en PG + Change Data Capture (CDC / PostgreSQL Logical Replication WAL → NATS)*.
* **Opción 4:** *Inversión Log-First (NATS JetStream como Fuente Primaria de Escritura y PostgreSQL como Proyección)*.

## Evaluación Comparativa de Alternativas

### Opción 1: RBAC Estricto por Esquema en PG + Patrón Transactional Outbox en SQL + Volumen Persistente JetStream
* **Ventajas:** Elimina tanto el *content coupling* (vía RBAC en PostgreSQL) como el problema de *dual-write* entre PostgreSQL y NATS mediante tablas `outbox_events` y un worker de despacho en proceso.
* **Desventajas:** Añade tablas `outbox_events` en los esquemas emisores y un bucle de polling/relay por unidad desde el MVP de Fase 0.
* **Impacto en Restricciones:** Costo monetario $0, pero añade complejidad de código y mantenimiento de tablas outbox en Fase 0.

### Opción 2: RBAC Estricto por Esquema en PG + Publicación Síncrona con Reintento en Memoria (`Backoff`) + Volumen Persistente JetStream (Sin Outbox) — propuesta
* **Ventajas:**
  * **Elimina de raíz el riesgo de *content coupling* (`DT-01`) desde Fase 0** a costo $0: PostgreSQL impone que `pater_app`, `filius_app` y `spiritus_app` solo puedan acceder a su propio esquema.
  * **Garantiza persistencia en disco de NATS JetStream (`DT-05`)** montando el volumen `nats_data:/data` con `-sd /data` en `docker-compose.yml`.
  * **Mantiene máxima simplicidad en el código Go (YAGNI):** el adaptador `EventPublisher` publica síncronamente tras el `COMMIT` local aplicando un retry en memoria con *Exponential Backoff + Jitter* (3 intentos: `100 ms`, `500 ms`, `2 s`), cubriendo micro-cortes de red sin requerir tablas `outbox_events` ni workers adicionales.
* **Desventajas:** Existe una ventana residual mínima si el proceso Go muere por `SIGKILL`/`OOM` exactamente en los milisegundos posteriores al `COMMIT` de PostgreSQL y previos al ack de NATS (o si NATS cae por más de 3 segundos), mitigada por logs estructurados de error y por el re-escaneo periódico en `Spiritus`.
* **Impacto en Restricciones:** Equilibrio óptimo entre robustez en producción y sobriedad operativa para un equipo de 3–5 desarrolladores.

### Opción 3: RBAC Estricto por Esquema en PG + Change Data Capture (CDC WAL → NATS)
* **Ventajas:** Garantía transaccional desde el WAL de PostgreSQL sin código de publicación en la aplicación.
* **Desventajas:** Requiere operar un proceso CDC externo (Debezium / conector WAL) y acopla la estructura interna de tablas SQL al contrato de eventos NATS (violando ADR-103).
* **Impacto en Restricciones:** Sobre-ingeniería incompatible con las restricciones de Fase 0.

### Opción 4: Inversión Log-First (NATS JetStream como Fuente Primaria y PostgreSQL como Proyección)
* **Ventajas:** Evita el *dual-write* escribiendo primero en el stream de NATS.
* **Desventajas:** Introduce consistencia eventual en lecturas transaccionales (*read-your-own-writes*) en la arista `Pater ↔ Filius` y convierte al nodo único de NATS en punto único de fallo para operaciones OLTP.
* **Impacto en Restricciones:** Contradice ADR-103 y eleva el riesgo en Fase 0.

## Decisión Adoptada y Justificación

Se elige la **Opción 2**, junto con la **reconciliación explícita del SLA de disponibilidad (`QA-8`)** por fase:

1. **Aislamiento RBAC en PostgreSQL desde Fase 0 (Mitigación de `DT-01`):** Se adelanta de Fase 2 a Fase 0 la creación de tres roles de base de datos con mínimo privilegio (`pater_app`, `filius_app`, `spiritus_app`). Se revocan los permisos de `PUBLIC` y cada unidad se conecta con su propio rol y `search_path`, teniendo permisos `USAGE, CREATE, SELECT, INSERT, UPDATE, DELETE` **exclusivamente** sobre su esquema.
2. **Persistencia de JetStream y Retry en Publicación (Mitigación de `DT-05`):**
   - El contenedor `nats` en `fase_0/deploy/docker-compose.yml` monta el volumen persistente `nats_data:/data` con `command: ["-js", "-sd", "/data", "-m", "8222"]`.
   - El adaptador `EventPublisher` en **Filius** y **Spiritus** ejecuta la publicación a JetStream tras el `COMMIT` local con **3 reintentos en memoria (*Exponential Backoff*: `100 ms`, `500 ms`, `2 s` + jitter $\pm 20\%$)**.
3. **Reconciliación Formal del Trade-off de Disponibilidad (`QA-8` vs `DT-03`):**
   - **Objetivo ideal del driver `QA-8`:** **99.99%** ($\le 5\text{ min/mes}$ de caída).
   - **SLO comprometido en Fase 0 (1 Droplet, ~$80/mes):** **99.9%** ($\approx 43.2\text{ min/mes}$ de caída máxima admitida). Esta diferencia se registra como trade-off consciente de la Iteración III/IV en `DT-03`, alcanzándose el **99.99%** en la Fase 2 al migrar a K3s multi-nodo con auto-healing.

## Consecuencias

* **Impacto Positivo:** El interés (riesgo activo) de `DT-01` y `DT-05` cae drásticamente antes de salir a producción en Fase 0, sin gastar un dólar adicional ni añadir procesos pesados; la documentación corrige la equivalencia matemática del SLO de Fase 0 (`99.9% = ~43.2 min/mes`).
* **Trade-offs y Riesgos Aceptados:** Se acepta la ausencia de tabla *Transactional Outbox* en Fase 0 a cambio de simplicidad de código; si se observaran pérdidas de eventos por caídas prolongadas de NATS, el patrón Outbox (Opción 1) o el cluster NATS de 3 nodos se activarán en Fase 1.
* **Deuda Arquitectónica / Acciones Futuras:** Actualizar `DT-01`, `DT-03` y `DT-05` en [deuda-arquitectonica.md](../registers/deuda-arquitectonica.md) para reflejar los controles ya activos en Fase 0.
