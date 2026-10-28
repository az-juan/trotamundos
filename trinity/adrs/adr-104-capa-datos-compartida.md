# ADR-104: Capa de datos compartida con segregación por esquema

* **Estado:** Propuesto (rama `main-trinity`)
* **Fecha:** 2026-10-28
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** II (persistencia y refinamiento)

## Contexto y Planteamiento del Problema

`main` adopta **sin base de datos compartida** (Newman ch.2, anti-pattern #1): cada microservicio es
dueño de sus tablas (ADR-004). Esa regla protege la información oculta cuando los servicios son
organizaciones independientes. Pero en la Trinidad las tres unidades **no son organizaciones
independientes**: son módulos de un mismo sistema, desplegados juntos, con un equipo pequeño. ¿Se
reintroduce la base de datos compartida (y cómo se evita la deriva a *content coupling*, Newman ch.2)?

## Drivers de Decisión

* **Constraints** — equipo pequeño, presupuesto: operar menos sistemas y menos instancias de DB.
* **QA-15 (analítica sin degradar lo transaccional)** — separar OLTP (PostgreSQL) de OLAP (ClickHouse).
* **QA-11 (modifiability)** — cada unidad dueña de su modelo, sin escribir en el interior de otra.
* **QA-9 (seguridad)** — mínimo privilegio sobre los datos.
* **ADR-004 (PostgreSQL + Redis + ClickHouse)** — stack heredado.

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *DB por unidad (como `main`)* — PostgreSQL con instancias/esquemas físicamente
  separados por unidad; cada unidad conecta solo a la suya.
* **Opción 2:** *Base de datos compartida sin reglas (acoplamiento libre)* — todas las unidades leen y
  escriben cualquier tabla. **(anti-pattern; descartada)**
* **Opción 3:** *Base de datos compartida con segregación por esquema + propiedad estricta* — un solo
  PostgreSQL, pero **un esquema por unidad** (`pater`, `filius`, `spiritus`); cada unidad es dueña de
  sus tablas y **solo lee** las de otras unidades vía contratos/proyecciones, nunca escribe en el
  esquema ajeno. **(propuesta)**
* **Opción 4:** *CQRS físico con DB separada para lectura analítica* — OLTP en PostgreSQL por unidad y
  OLAP en ClickHouse; sin compartir OLTP. (El OLAP ya está en ADR-004; no resuelve el costo del OLTP
  compartido en Fase 0.)

## Evaluación Comparativa de Alternativas

### Opción 1: DB por unidad (como main)
* **Ventajas:** máximo aislamiento de datos (Newman ch.2). **Desventajas:** 3 instancias de PostgreSQL
  (o 3 esquemas con DBs separadas) → 3× costo y operación en Fase 0; para un equipo pequeño es
  desproporcionado. **Restricciones:** choca con presupuesto.

### Opción 2: DB compartida sin reglas
* **Ventajas:** ninguna complejidad de conexión. **Desventajas:** *content coupling* (Newman ch.2): las
  unidades asumen la estructura interna unas de otras → erosión y deuda arquitectónica (Bass ch.23).
  **Restricciones:** inaceptable.

### Opción 3: DB compartida con segregación por esquema — propuesta
* **Ventajas:** **un** PostgreSQL (costo y operación mínimos); cada unidad dueña de su esquema
  (`pater`/`filius`/`spiritus`); las relaciones inter-unidad se hacen por **contrato** (evento NATS o
  read-model), no por escribir en la tabla ajena → se conserva la *information hiding* (Parnas) sin
  multiplicar instancias. **Desventajas:** el aislamiento de fallos de DB es menor que DBs físicas
  (mitigado: PostgreSQL managed con HA de DO). **Restricciones:** óptimo para equipo pequeño.

### Opción 4: CQRS físico con DB separada
* **Ventajas:** separa claramente lectura analítica de transaccional. **Desventajas:** ClickHouse ya lo
  hace (ADR-004/ADR-010); no justifica 3 PostgreSQL en Fase 0. **Restricciones:** sobrecarga.

## Decisión Adoptada y Justificación

Se elige la **Opción 3**. La regla de Newman «sin DB compartida» nace para servicios que son
**organizaciones independientes** (Conway, ch.15); aquí las unidades son **módulos de un mismo sistema**
con un equipo único, así que el riesgo real no es la guerra de organizaciones sino la **deriva a
content coupling**. La segregación por esquema + propiedad estricta (cada unidad escribe solo en su
esquema; lo que necesita de otra unidad viene por evento o read-model) mitiga ese riesgo manteniendo un
solo PostgreSQL. ClickHouse sigue **exclusivo de Spiritus** (OLAP), preservando QA-15. Redis es caché
compartido (sesiones, rate limit, dedup, idempotencia) porque no contiene datos de dominio con
propietario único.

| Almacén | Propiedad en la Trinidad |
|---------|--------------------------|
| **PostgreSQL** (managed) | Compartido; esquemas `pater` / `filius` / `spiritus`; escritura por dueño |
| **Redis** (managed) | Caché compartido (sin datos de dominio con dueño único) |
| **ClickHouse** (Droplet 2, Fase 1) | Exclusivo de **Spiritus** (OLAP) |

## Consecuencias

* **Impacto Positivo:** un solo PostgreSQL (costo/operación mínimos); información oculta preservada por
  esquema; analítica aislada del OLTP; migración limpia a DBs separadas en Fase 2 si el escalado lo
  exige (los puertos hexagonales lo absorben, ADR-002).
* **Trade-offs y Riesgos Aceptados:** una caída de PostgreSQL afecta a las 3 unidades (mitigado: managed
  con HA); tentación de escribir entre esquemas → prohibido por convención + revisión (y, más adelante,
  control de acceso por rol de DB).
* **Deuda Arquitectónica / Acciones Futuras:** si una unidad crece hasta merecer su propia DB, extraer
  su esquema a una instancia dedicada sin tocar el dominio (regla 2-de-3, ADR-102).
