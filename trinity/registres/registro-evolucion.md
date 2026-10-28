# Registro de evolución — Trinidad

Changelog cronológico de los cambios de documentación de la arquitectura **Trinidad** (rama
`main-trinity`). Formato: **fecha · tipo · artefacto · motivo · trazabilidad · commit**.

> Convención (ver [README.md](./README.md)): un cambio que contradiga un ADR aceptado exige primero un
> ADR de revisión (Regla 2 de la SPEC). Ninguna entrada de aquí altera una decisión sin su ADR.

---

## 2026-10-28 — Documentación inicial de la Trinidad

- **Tipo:** creación (iteración ADD I–IV, re-elección del estilo macro).
- **Artefactos creados:**
  - `README.md` — idea central, el triángulo, encaje en SDD (CON-1/CON-5).
  - `tecnologias.md` — stack adaptado (sin cambios de lenguaje/framework/almacén; re-partición en 3 unidades).
  - `adrs/adr-101..105` — Trinidad (3 unidades), límites de unidad, comunicación inter-unidad, capa de datos compartida, despliegue.
  - `architecture/vista_modulos.md`, `vista_cyc.md`, `vista_despliegue.md` — las tres vistas.
- **Motivo:** proponer un estilo macro que conserve la separación por perfil de carga (batch vs per-request) **sin** el overhead de 7 microservicios, respetando las restricciones (equipo pequeño, poco tiempo, presupuesto).
- **Trazabilidad:** SPEC-001 (ADD 3.0 + SDD, CON-3); drivers `constraints.md`, QA-1/QA-3/QA-6/QA-7/QA-15.
- **Commit:** `e20804e`.

---

## 2026-10-28 — Refuerzo de la estructura triangular en las vistas

- **Tipo:** mejora de vistas (consistencia visual del motivo triangular).
- **Artefactos:**
  - `architecture/vista_modulos.md` — **añadido** el diagrama de referencia *«triángulo trinitario»* (las 3 unidades conectadas dos a dos con su conector/contrato por arista) + tabla de aristas + nota de *information hiding*; aclarada la leyenda de color del triángulo. **(Era el vacío real: la vista era un árbol `go.work → unidades → shared` sin el triángulo.)**
  - `architecture/vista_cyc.md` y `vista_despliegue.md` — nota de referencia cruzada al triángulo de referencia.
- **Motivo:** el usuario pidió ilustrar bien la estructura triangular en las tres vistas (código modular, runtime, despliegue).
- **Trazabilidad:** Bass ch.1/22 (tres categorías de estructuras); ADR-103 (aristas); ADR-104 (segregación).
- **Verificación:** 13 bloques mermaid validados con el parser real (12 → 13, todos OK); sin tablas corruptas.
- **Commit:** `6d8137c`.

---

## 2026-10-28 — Reorganización como arquitectura elegida + detalle por unidad + registro

- **Tipo:** reorganización estructural + nueva documentación de detalle + nuevo directorio de gobernanza.
- **Artefactos:**
  - `README.md` — **reenmarcado**: de «propuesta alternativa» a **arquitectura elegida**; título y tabla de documentación actualizadas con `units/` y `registres/`.
  - `adrs/adr-101..105` — estatus **Propuesto → Aceptado** (refleja la elección formal; la justificación y las 3+ opciones se conservan, SPEC §5).
  - `units/` **(nuevo)** — detalle por unidad, análogo a `../microservices/01..09`:
    - `README.md` — índice, triángulo de unidades, trazabilidad drivers → unidades.
    - `01-arquitectura-global.md` — monorepo `go.work` + 3 `go.mod`, hexagonal, `shared/`, triángulo de conectores, cross-cutting.
    - `02-pater-facade.md` — Gateway · Web BFF · Mobile BFF.
    - `03-filius-nucleo.md` — Itinerary · Integration.
    - `04-spiritus-mente.md` — Email Ingest · Analytics.
    - `05-catalogo-eventos.md` — eventos NATS **restringidos a la arista Filius↔Spiritus**.
  - `registres/` **(nuevo)** — registro de evolución: `README.md`, `registro-evolucion.md` (este archivo), `deuda-arquitectonica.md`, `transiciones.md`.
- **Motivo:** presentar a Trinity como la arquitectura escogida, mantener los ADRs y drivers adecuados, y registrar la evolución conforme a la SPEC (Paso 8).
- **Trazabilidad:** SPEC-001 §5 (MADR con ≥3 opciones), Paso 8 (cierre de gobernanza), CON-3 (vistas + ADRs), Regla 2 (coherencia).
- **Verificación:** todos los bloques mermaid nuevos validados con el parser real; sin tablas corruptas.
- **Commit:** *(pendiente de commit en esta sesión)*.

---

## 2026-10-28 — Consolidación: Trinity como documentación única del repositorio

- **Tipo:** reorganización estructural + migración de drivers/ADRs + eliminación de docs de microservicios.
- **Artefactos:**
  - `drivers/` → **movido a `trinity/drivers/`** (requerimientos, atributos, constraints, concerns).
  - `adrs/adr-001..014` → **movidos a `trinity/adrs/`** con nombre descriptivo y **adaptados a Trinidad**:
    - **ADR-001** (7 microservicios) → estatus **Superado por ADR-101**; los bounded contexts y la regla 2-de-3 se heredan como módulos/unidades.
    - **ADR-002/003/004/005/006/007/008/009/010/011/012/013/014** → **Aceptados** + nota «Adaptación a Trinidad» (hexagonal por módulo, Go/go.work, PG con esquemas, NATS a una arista, Chi/BFFs en Pater, email/analytics en Spiritus, 3 imágenes, JWT del Gateway de Pater).
  - **Eliminados:** `microservices/` (01..09) y `architecture/` raíz (vistas de microservicios) — superados por `trinity/units/` y `trinity/architecture/`.
  - `README.md` raíz → reescrito como **página de entrada** a la documentación de Trinidad.
- **Motivo:** presentar a Trinidad como la única documentación de arquitectura del repositorio; mantener los ADRs y drivers adecuados y reconciliados con la Trinidad.
- **Trazabilidad:** SPEC-001 Regla 2 (coherencia; ADR-001 superado formalmente por ADR-101, no borrado sin registro), CON-3.
- **Nota:** los enlaces funcionales de `trinity/` son todos internos (`./…`); las referencias `../architecture/X` y `../microservices/X` que quedan son texto descriptivo de la rama `main` (columna «Equivalente en `main`»), no enlaces rotos.
- **Commit:** *(pendiente de commit en esta sesión)*.

---

## Futuro (borrador — anotar al concretar)

- Extracción de **Notification** como módulo de Spiritus (Fase 1) → actualizar `units/04` y `05`.
- Extracción de **Identity** (Fase 2, ADR-104) → Pater deja de emitir JWT; nuevo ADR de revisión.
- Migración de módulo→servicio por regla 2-de-3 (ADR-102) → anotar ADR + mover módulo a `services/`.
- Migración NATS→Kafka si se cumple 2-de-3 (ADR-103) → solo cambia el adaptador de la arista.
