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
  - `README.md` — **reenmarcado**: de «propuesta alternativa» a **arquitectura elegida**; título y tabla de documentación actualizadas con `units/` y `registers/`.
  - `adrs/adr-101..105` — estatus **Propuesto → Aceptado** (refleja la elección formal; la justificación y las 3+ opciones se conservan, SPEC §5).
  - `units/` **(nuevo)** — detalle por unidad, análogo a `../microservices/01..09`:
    - `README.md` — índice, triángulo de unidades, trazabilidad drivers → unidades.
    - `01-arquitectura-global.md` — monorepo `go.work` + 3 `go.mod`, hexagonal, `shared/`, triángulo de conectores, cross-cutting.
    - `02-pater-facade.md` — Gateway · Web BFF · Mobile BFF.
    - `03-filius-nucleo.md` — Itinerary · Integration.
    - `04-spiritus-mente.md` — Email Ingest · Analytics.
    - `05-catalogo-eventos.md` — eventos NATS **restringidos a la arista Filius↔Spiritus**.
  - `registers/` **(nuevo)** — registro de evolución: `README.md`, `registro-evolucion.md` (este archivo), `deuda-arquitectonica.md`, `transiciones.md`.
- **Motivo:** presentar a Trinity como la arquitectura escogida, mantener los ADRs y drivers adecuados, y registrar la evolución conforme a la SPEC (Paso 8).
- **Trazabilidad:** SPEC-001 §5 (MADR con ≥3 opciones), Paso 8 (cierre de gobernanza), CON-3 (vistas + ADRs), Regla 2 (coherencia).
- **Verificación:** todos los bloques mermaid nuevos validados con el parser real; sin tablas corruptas.
- **Commit:** `a3a8364` (creación de `units/` y re-marca de la Trinidad como arquitectura elegida).

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
- **Commit:** `3b95642` (main-trinity; fast-forward a `main` en `d919297`).

---

## 2026-10-28 — Renombrado del registro y consolidación en main
- `registres/` → `registers/` (terminología inglesa; el propio registro se renombra). Referencias actualizadas en `README` raíz, `trinity/README` y `units/01`.
- `main` actualizado a Trinidad por **fast-forward** desde `main-trinity` (sin conflictos; `main` = documentación única de Trinidad).
- Verificación: 0 enlaces rotos internos, 0 referencias colgantes a carpetas borradas, 26 bloques mermaid válidos, sin "registres" suelto.
- **Estado: listo para empezar la implementación.**

---

## 2026-10-28 — Resolución de temas no tratados y refinamiento para la primera iteración de implementación
- **Tipo:** resolución de gaps + refinamiento (post-revisión completa de consistencia).
- **Artefactos nuevos:**
  - `fase_0/docs/api/openapi.yaml` — OpenAPI 3.1 (auth, trips, bookings, sharing, insights, nearby/GPS, health, envelope de errores).
  - `fase_0/docs/events/events.yaml` — JSON Schema de los 6 eventos NATS + streams, retención, idempotencia, migración a Kafka.
  - `trinity/data/modelo-datos.md` — ERD, DDL por esquema (pater/filius/spiritus), state machine del Booking, ClickHouse (Fase 1), migraciones.
  - `trinity/dev/setup-local.md` — guía de desarrollo local Fase 0.
  - `fase_0/deploy/docker-compose.yml` — materializado con healthchecks.
  - `fase_0/.env.example` — variables por unidad.
  - `trinity/testing/estrategia-testing.md` — pirámide, contratos, flaky, cobertura.
  - `trinity/ops/operaciones.md` — observabilidad (métricas), backup/restore, privacidad, runbook.
- **Refinamientos aplicados:**
  - `constraints.md` y `design-purpose.md` expandidos (equipo, tiempo, presupuesto, YAGNI, disparadores, GPS en scope).
  - `units/02` (Pater): flujo JWT RS256, protocolo SSE, error envelope, geolocalización (GPS, QA-13).
  - `units/03` (Filius): contratos de proveedores (integration) con cadena de resiliencia.
  - `units/01`: métricas Prometheus, variables de entorno, error envelope.
  - `atributos.md` QA-13: GPS con permiso explícito, solo foreground.
  - `vista_cyc.md`: claridad en re-publicación de `booking.discovered`.
  - `vista_modulos.md`: leyenda de color clarificada.
  - `ADR-001..014`: encabezado de procedencia (decisión vigente = ADR-101..105).
  - `transiciones.md`: criterio de salida explícito para Iteración II.
  - `deuda-arquitectonica.md` DT-04: referencia simplificada.
  - `trinity/README.md`: tabla de documentación actualizada.
- **Validación:** 0 enlaces rotos (40), 28/28 mermaid OK, 0 pipes doblados, sin `registres` suelto.
- **Trazabilidad:** SPEC-001 Paso 8 (cierre de gobernanza), Regla 2 (coherencia), CON-3 (vistas + ADRs), CON-4 (prototipo).
- **Commit:** *(este commit)*.

---

## 2026-11-21 — Hardening de Fase 0, reducción de deuda arquitectónica y cierre de infraestructura operativa (ADR-106..108)

- **Tipo:** nuevas decisiones MADR de Iteración IV + refinamiento de modelo de datos/contratos + materialización de infraestructura de `fase_0/deploy/`.
- **Artefactos nuevos:**
  - `trinity/adrs/adr-106-identidad-y-preferencias-fase0.md` — propiedad de `filius.users` (con `preferences JSONB` indexado por GIN) en Filius y validación de credenciales vía contrato REST interno `Pater → Filius` (`POST /internal/auth/verify`).
  - `trinity/adrs/adr-107-persistencia-analitica-fase0.md` — polimorfismo hexagonal en el puerto `AnalyticsRepository` de Spiritus: adaptador PostgreSQL (`spiritus.analytics_events` + `spiritus.insight_aggregates`) en Fase 0 → adaptador ClickHouse en Fase 1.
  - `trinity/adrs/adr-108-aislamiento-rbac-y-resiliencia-eventos.md` — aislamiento RBAC por esquema desde Fase 0 (`pater_app`, `filius_app`, `spiritus_app`), persistencia JetStream en volumen `nats_data` con retry `Exponential Backoff` en memoria, y reconciliación formal del SLA de `QA-8` (99.9% ≈ 43.2 min/mes en Fase 0 vs. 99.99% ≤ 5 min/mes en Fase 2).
  - `fase_0/deploy/postgres/init.sql` — DDL completo con los 3 esquemas, roles RBAC, tablas, índices y usuario seed.
  - `fase_0/deploy/nginx/conf.d/default.conf` — proxy hacia Pater con soporte SSE (`proxy_buffering off` en `/insights/stream`) y preservación de firmas HMAC en `/webhooks/`.
  - `fase_0/deploy/Dockerfile` — build multi-stage Go 1.23 (`CGO_ENABLED=0`) con targets `pater`, `filius` y `spiritus`.
- **Artefactos actualizados:**
  - `trinity/data/modelo-datos.md` — ERD y DDL actualizados con `filius.users`, coordenadas `lat`/`lon` en `filius.bookings`, tablas analíticas de Fase 0 en `spiritus` y roles RBAC.
  - `fase_0/docs/api/openapi.yaml` — añadidos `/auth/refresh`, `/auth/logout`, `/users/me/preferences` (`GET`/`PATCH`), `/mailboxes` (`GET`/`POST`), `/whitelists` (`GET`/`POST`/`DELETE`) y `/webhooks/{provider}` (`POST`).
  - `fase_0/docs/events/events.yaml` — enriquecidos `BookingConfirmed` y `BookingCancelled` con `user_id`, `type`, `destination` y `segments`; documentada la política `publisher_resilience`.
  - `fase_0/deploy/docker-compose.yml` y `fase_0/.env.example` — volumen `nats_data:/data` (`-sd /data`), roles RBAC por unidad y `SPIRITUS_ANALYTICS_DRIVER=postgres`.
  - `README.md`, `trinity/README.md`, `trinity/units/01..05`, `trinity/architecture/vista_cyc.md`, `trinity/architecture/vista_despliegue.md`, `trinity/drivers/constraints.md`, `trinity/ops/operaciones.md`, `trinity/dev/setup-local.md`, `trinity/registers/deuda-arquitectonica.md` y `trinity/registers/transiciones.md` — sincronizados con `ADR-106..108`.
- **Trazabilidad:** SPEC-001 Regla 2 (coherencia), Regla 3 (sobriedad operativa), Regla 4 (MADR ≥ 3 opciones), Paso 8 (gobernanza y reducción de deuda `DT-01`, `DT-03`, `DT-05`).

---

## 2026-11-22 — Asistente Virtual (FR-10/FR-11, QA-16/QA-17): auditoría de consistencia, ADR-109/110 y propagación a vistas y contratos

- **Tipo:** ampliación de la Iteración IV (CON-5: *More decisions and improvements*) con destino **Fase 1**; nueva decisión MADR + actualización de drivers, vistas, unidades, contratos y registros.
- **Auditoría previa (SPEC-001 Regla 2 / directiva 1, Paso 0):** se detectó que `drivers/design-purpose.md` §5 excluía «chat» del scope de **Fase 0** y que `constraints.md` prohíbe plataformas gestionadas con presupuesto ~$80/mes. **Se advirtió el conflicto antes de documentar** y se resolvió **sin revocar** la decisión: el Asistente Virtual se provee en **Fase 1**, con Fase 0 intacta. Se reutiliza FR-6 (deep-link) para la derivación, se mantiene NATS en una sola arista (ADR-103) y no se añade un cuarto desplegable (ADR-101/102).
- **Artefactos nuevos:**
  - `adrs/adr-109-asistente-virtual.md` — módulo `assistant/` en Pater, puerto `IntentClassifier` (LLM con ACL + redacción de PII; fallback determinista), streaming SSE/htmx; 4 alternativas evaluadas.
  - `adrs/adr-110-derivacion-y-estado-asistente.md` — agregado `AssistanceCase` en Filius, `context_token` en Redis, deep-link a la agencia preferida y eventos `assistance.*`; 4 alternativas evaluadas.
- **Drivers actualizados:** `drivers/requerimientos.md` (FR-10, FR-11), `drivers/atributos.md` (QA-16, QA-17), `drivers/design-purpose.md` (capacidad, actor agencia, alcance Fase 1, métricas), `drivers/constraints.md` (YAGNI Fase 0 + disparador Fase 1).
- **Vistas y detalle actualizados:** `architecture/vista_modulos.md`, `vista_cyc.md` (escenario 6), `vista_despliegue.md`; `units/01..05` (nuevos módulos, caso, evento `assistance.>`); `README.md` raíz y `trinity/README.md`.
- **Contratos e infra actualizados:** `fase_0/docs/api/openapi.yaml` (`/assistant/*` + `/internal/assistance/*`, schemas, `preferred_agency_id`), `fase_0/docs/events/events.yaml` (2 eventos + stream `ASSISTANCE`), `trinity/data/modelo-datos.md` (ERD, DDL y state machine de `AssistanceCase`), `fase_0/deploy/postgres/init.sql`, `nginx/conf.d/default.conf` (SSE del chat), `.env.example` (flags Fase 1).
- **Gobernanza actualizada:** `registers/deuda-arquitectonica.md` (**DT-06**: proveedor LLM), `registers/transiciones.md` (criterio Fase 1), `ops/operaciones.md` (métricas/alertas/privacidad/runbook), `testing/estrategia-testing.md` (LLM fake, redacción de PII, fallback).
- **Trazabilidad:** SPEC-001 Regla 2, Regla 3, Regla 4 (≥ 3 opciones), Paso 8; Bass ch.19/20/22/23; Keeling (HART, ASRs, decision matrix, ADRs); Newman ch.2/4/6/14 (coupling, estilos de comunicación, sagas, BFF).
- **Estado:** documentación lista; implementación del prototipo (CON-4) y activación del feature flag pendientes.

---

## 2026-11-22 — Prototipo ejecutable Fase 0 (CON-4): tres unidades, vistas templ y diseño Buenos Aires 1930

- **Tipo:** construcción del prototipo y cierre de gobernanza (Paso 8), sin cambios arquitectónicos: se materializa lo ya decidido en ADR-101..108.
- **Artefactos nuevos:** `go.work`, `shared/event`, `units/filius` (dominio + store + API REST + tests), `units/spiritus` (parser de correo + analytics + API + tests), `units/pater` (clientes, handlers, vistas `*.templ` + CSS + HTMX local + tests), `scripts/run.sh` y `scripts/test.sh`.
- **Interfaz:** HTML generado por templ, `base.css` con CSS Grid/Flexbox y paleta de `images/palette.jpg`; HTMX servido localmente, **sin JavaScript propio**. Filtros y formularios usan `hx-get`/`hx-post` con swaps OOB.
- **Verificación real:** `./scripts/test.sh` en verde (4 módulos), `healthz` de las 3 unidades, assets 200, flujo completo probado por HTTP y en navegador (filtro activo, crear viaje, agregar reserva, simular correo → `booking.discovered`/`booking.confirmed`).
- **Deuda registrada:** `DT-07` (adaptadores en memoria, bus HTTP en vez de NATS) para conectar la infra de `fase_0/deploy/` sin tocar dominio.
- **Trazabilidad:** SPEC-001 CON-4 / Paso 8; Bass ch.22 (vistas) y ch.23 (deuda); Newman ch.2 (information hiding) y ch.14 (BFF).

---

## 2026-11-23 — Completar Fase 0: páginas, componentes templ y despliegue Docker Compose

- **Tipo:** continuación del prototipo ejecutable (CON-4) y cierre de gobernanza (Paso 8), sin cambiar ADRs.
- **Artefactos nuevos:** `units/pater/views/components.templ` (biblioteca de componentes), `views/pages.templ` (10 páginas), `internal/web/auth.go` (JWT RS256 + JWKS + cookie), `units/filius/internal/store/extras.go` y `internal/api/extras.go` (usuarios, preferencias, compartir, viajeros, nearby, webhooks HMAC), `units/spiritus/internal/mail/` (buzones, whitelist, dedup SHA256) y `internal/api/extras.go` (SSE).
- **Integración:** Pater consume las aristas REST de Filius y Spiritus; Filius publica hechos a Spiritus por HTTP en el prototipo; webhooks HMAC verificados; `/api/events/stream` disponible para clientes SSE; navegación HTMX con `hx-push-url` y swaps OOB.
- **Despliegue:** `docker compose up -d --build` verificado con los 7 servicios healthy; interfaz servida por Nginx en `:8088`; builder actualizado a Go 1.25 (requisito de templ v0.3.1020, dentro de «Go 1.23+»). Puertos host configurables para evitar colisiones.
- **Verificación real:** `./scripts/test.sh` en verde (4 módulos), `go vet`, YAML y `git diff --check` limpios; páginas, estáticos, JWKS, login y flujo completo comprobados por HTTP y en navegador dentro de los contenedores.
- **Trazabilidad:** SPEC-001 CON-4 / Paso 8 / Regla 3; Bass ch.15/22; Keeling (componentes, vistas, ADRs); Newman ch.5 (information hiding), ch.7 (build-once), ch.14 (BFF).

---

## 2026-11-23 — Auditoría de coherencia Fase 0 y cierre de brechas (API JSON, NATS, proxy SSE/webhooks)

- **Tipo:** auditoría de consistencia (SPEC-001 Regla 2 / Paso 0) y corrección de desvíos del prototipo, sin cambiar ADRs.
- **Desvíos detectados:** (1) Pater no exponía el API JSON del OpenAPI para el Mobile BFF (ADR-013): faltaban `/auth/refresh`, `/users/me/preferences`, `/insights/stream`, `/webhooks/{provider}` y las variantes JSON con `snake_case`; (2) los eventos viajaban por HTTP aunque ADR-103/108 exigen NATS JetStream y el compose ya levantaba el broker; (3) FR-5 no tenía el job de auto-archivado active; (4) no había proxy de webhooks firmados en el borde.
- **Correcciones:** `units/pater/internal/web/api.go` (negociación por `Accept: application/json`, JWT refresh, preferencias, SSE y proxy HMAC); `shared/event/nats.go` (streams `TRIPS/BOOKINGS/NOTIFICATIONS/ASSISTANCE`, publisher y consumidor durable); `cmd/filius` y `cmd/spiritus` usan NATS cuando `NATS_URL` está definida, con fallback HTTP; job de auto-archivado en Filius; `docker-compose.yml` inyecta `NATS_URL` a ambas unidades.
- **Contratos:** los tipos de vista llevan tags JSON `snake_case`, alineados con `fase_0/docs/api/openapi.yaml`; se mantienen las rutas HTML/HTMX para el navegador.
- **Verificación:** `./scripts/test.sh` en verde (incluye tests de la API JSON, refresh/preferencias, proxy HMAC y fallbacks), `go vet`, YAML y `git diff --check` limpios. En contenedores se comprobó que NATS entrega `booking.discovered`, `booking.confirmed` y `trip.disrupted` al read-model, que `/insights/stream` hace streaming SSE y que el webhook HMAC actualiza el estado de la reserva.
- **Defectos encontrados y corregidos durante la verificación:** el cliente enviaba `POST` donde Filius exige `PATCH`; `main.go` no inyectaba `FILIUS_URL`/`SPIRITUS_URL` en los proxies; `io.Copy` no liberaba el buffer del SSE; un PATCH parcial de preferencias sobrescribía campos no enviados. Además se cachearon los módulos Go en el Dockerfile para acelerar rebuilds.
- **Trazabilidad:** SPEC-001 Regla 2 y Paso 8; ADR-006/013/014 (Gateway, BFF, JWT), ADR-103/107/108 (NATS y read-model), FR-5/FR-8/QA-9/QA-13; Bass ch.15 (interfaces) y ch.22 (documentación).

---

## 2026-11-23 — Eliminación de la API JSON: sqlc directo, HTMX y sesión por cookie (ADR-111)

- **Tipo:** revisión del prototipo Fase 0 (CON-4) formalizada en ADR-111, con ≥ 3 alternativas evaluadas y tabla de conflictos del Paso 0.
- **Artefactos nuevos:** `trinity/adrs/adr-111-sin-json-sqlc-htmx.md`, `sqlc.yaml`, `db/schema.sql`, `db/queries/{trips,users,mail}.sql`, `shared/db/` (código generado), `units/pater/internal/logic/logic.go`, `units/pater/internal/web/session.go`.
- **Cambios:** eliminados `api.go`, `api_test.go`, `client.go` y `auth.go` de Pater; los handlers usan un puerto `Store` implementado por sqlc; `PATER_PG_DSN` es obligatorio; `init.sql` añade grants y datos semilla; el compose inyecta el DSN.
- **Contratos y documentación:** `fase_0/docs/api/openapi.yaml` pasa a contrato de referencia diferido a Fase 1; README, `setup-local.md`, `design-purpose.md`, `constraints.md`, `units/02`, `testing`, `ops`, deuda (`DT-08`, `DT-09`), transiciones e índice de ADRs actualizados.
- **Deuda asumida:** sin BFF móvil/JWT en Fase 0; `pater_app` accede a esquemas ajenos en el prototipo; se restaura en Fase 1.
- **Verificación:** `sqlc generate` exitoso; tests de Pater con fake `Store` en verde; en contenedor se comprobó SSR desde PostgreSQL, login con cookie HttpOnly, alta de viaje y preferencias persistidas con sqlc, ingesta demo, 404 en rutas JSON y navegación HTMX sin JS propio. Durante la verificación se corrigieron los grants de esquema/USAGE y el acceso a `pgcrypto.crypt`. Finalizada el 2026-11-23.

---

## 2026-11-23 — Revisión de coherencia por iteración ADD y ajuste de vistas

| Iteración (CON-5) | Documentación | Implementación Fase 0 | Discrepancia detectada | Acción |
|-------------------|---------------|-----------------------|------------------------|--------|
| **I** — estilo de referencia | Trinidad, 3 unidades, triángulo (ADR-101/102) | 3 deployables en `docker compose`, NATS solo `Filius↔Spiritus` | Las vistas seguían rotulando `Pater↔Filius` como REST activo y `Spiritus→Pater` como SSE | Etiquetas pasan a «referencia Fase 1» con nota ADR-111 en `vista_modulos`, `vista_cyc` y `vista_despliegue` |
| **II** — funcionalidad y tecnología | OpenAPI 3.1, modelo de datos, RBAC por esquema (ADR-104/106/108) | sqlc sobre PostgreSQL real; grants ampliados para `pater_app` | OpenAPI parecía superficie implementada; RBAC estricto no regía en el prototipo | OpenAPI marcado como contrato diferido; `DT-08`/`DT-09` registran la desviación |
| **III** — trade-offs QA | REST síncrono, JWT RS256, correlación, SSE (ADR-103/014) | Cookie firmada, HTMX, `X-Correlation-ID`, consultas sqlc | Los diagramas de secuencia validaban JWT/JWKS y empujaban SSE al navegador | Escenario 2 usa cookie; SSE queda como referencia Fase 1; `QA-9` se cumple con HttpOnly |
| **IV** — mejoras y prototipo | 10 páginas, componentes templ, CON-4 | SSR desde PostgreSQL, sin JS propio, sesión por cookie | Sin discrepancias funcionales; documentación de UI actualizada en ADR-111 | Sin cambios de código, solo sincronización documental |

- **Conclusión:** no hay contradicciones abiertas entre código y documentación para Fase 0. Las decisiones de canal móvil/JWT y RBAC estricto quedan diferidas y trazadas en deuda.

---

## 2026-11-23 — Revisión por iteración, tonos madera y mapa SSR

- **Tipo:** verificación de coherencia Iteración I–IV (tabla en esta sección) y ampliación visual/functional del prototipo.
- **Cambios:** vistas `vista_modulos`, `vista_cyc` y `vista_despliegue` dejan los conectores REST/JWT/SSE como referencia Fase 1 y muestran el acceso sqlc real; `base.css` incorpora tonos madera/café (`--wood`, `--wood-dark`, `--wood-light`, `--coffee`) y textura de veta; nueva consulta `ListGeoPoints`, método `Store.GeoPoints`, página `/map` con SVG generado por templ (río, calles, parques, marcadores y trazas) y enlace en la navegación.
- **QA-13:** el atributo de geolocalización pasa a describir el mapa SSR además de la búsqueda por coordenadas, sin JavaScript.
- **Verificación:** `sqlc generate`, `templ generate`, tests, `go vet` y `git diff --check` limpios. En contenedor se comprobó `/map` (200, 3 marcadores, 2 trazas, único `<script>` de HTMX) y se revisó visualmente la estética café 1930; se reforzó el contraste de río, calles y trazas.

---

## 2026-11-23 — Mapa real con go-staticmaps y limpieza de artefactos

- **Tipo:** decisión arquitectónica (ADR-112), evolución del prototipo Fase 0 y auditoría de obsolescencia.
- **Viaje de prueba renombrado:** la fila creada en la verificación como «Viaje desde sqlc» pasa a **«Buenos Aires → Bariloche»** (lugar real).
- **Mapa:** se sustituye el SVG dibujado a mano por `internal/maps.Renderer` (`go-staticmaps` + teselas OpenStreetMap). `/map/image` sirve un PNG real con marcadores, traza, caché de teselas y User-Agent; el marco madera y el filtro sepia mantienen la estética 1930. ADR-112 reescrito (antes describía SVG) y referencias actualizadas.
- **Artefactos eliminados:** `views.MapSVG`, helper `MapX/MapY`, estilos `.map-svg/.map-pin/.map-route`, variables JWT/JWKS y URLs internas de Pater en `.env.example`, JWKS en README/tabla de puertos y `setup-local`, credenciales JWT en `vista_despliegue`. Las secciones de `units/02`, `units/01`, `units/README`, `units/03` y `vista_modulos` que describían REST/JWT/Mobile BFF como Fase 0 quedan marcadas como **referencia Fase 1**; los ADR históricos se conservan como registro.
- **Verificación:** `sqlc generate`, `templ generate`, `go work sync`, tests, `go vet`, YAML/Mermaid/enlaces y `git diff --check` limpios. En contenedor: `/map/image` devuelve `image/png` real de 880×480 con 3 pines y traza dorada sobre calles de Buenos Aires; las coordenadas semilla se separaron para que los tres marcadores no se superpusieran; viaje renombrado visible en la portada; único `<script>` de HTMX.

---

## 2026-11-23 — Auditoría de completitud Fase 0 y E2E con Hurl

| Requerimiento | Implementación Fase 0 | Prueba Hurl / Go | Estado |
|---------------|------------------------|------------------|--------|
| FR-1 reservas de terceros | sqlc sobre `filius.bookings`, webhooks HMAC en Filius | `crud.hurl`, `integracion.hurl` | Completo |
| FR-2 captura por email | demo ingest y eventos por NATS | `htmx.hurl` | Completo (modo demo) |
| FR-3 whitelist | `/whitelists` + `spiritus.senders` | tests Go + `pages.hurl` | Completo |
| FR-4 gestión manual | `/trips`, `/bookings` | `crud.hurl` | Completo |
| FR-5 agrupación/archivado | `/trips/{id}/archive` | `crud.hurl` | Completo |
| FR-6 deep-linking | `views.DeepLinks` en detalle y compartido | `crud.hurl` | Completo |
| FR-7 analítica | `/insights` + `spiritus.analytics` | `htmx.hurl` | Completo |
| FR-8 compartir | `/trips/{id}/share`, `/shared/{token}` | `crud.hurl` | Completo |
| FR-9 monetización | insights agregados en el boletín | `pages.hurl` | Básico (Fase 1 comercial) |
| FR-10/FR-11 asistente | ADR-109/110, Fase 1 | no aplica | Diferido documentado |
| FR-12 mapa | `/map` + `/map/image` con go-staticmaps | `mapa.hurl` | Completo |
| QA-9 sesión | cookie HMAC HttpOnly, sin JWT/JWKS | `auth-preferences.hurl`, `smoke.hurl` | Completo |
| QA-13 geolocalización | `/nearby` y mapa real | `pages.hurl`, `mapa.hurl` | Completo |

- **Discrepancia encontrada:** Nginx enrutaba `/webhooks/` a Pater (ADR-111 ya no expone ese handler) y mantenía locations SSE de Fase 1. Se añadió `filius_backend`, se redirigieron los webhooks a Filius y se retiraron los locations SSE; `nginx -t` pasó y sin firma responde 401.
- **Hurl:** 7 archivos, **38 peticiones, 100% en verde en dos corridas consecutivas** (idempotente). `./scripts/hurl.sh` calcula HMAC, obtiene IDs/tokens de PostgreSQL y preserva el estado de `PNR-1930`.
- **Documentación corregida:** `testing/estrategia-testing.md` (sección 4bis, OpenAPI/Pact como Fase 1, mapa real), README (comando Hurl, roles de Filius/Spiritus, página Mapa) y `dev/setup-local.md` (migración real vía `init.sql`, Pater depende de PostgreSQL).

---

## 2026-11-23 — Diagnóstico de conexión y consistencia de bloques HTMX

- **`failed to connect: 127.0.0.1:55432 connection refused`:** no era un error de configuración. `docker ps` estaba vacío: los 7 contenedores se habían detenido, por lo que el puerto host publicado por `POSTGRES_HOST_PORT` no existía. Tras `docker compose up -d` el puerto quedó abierto. Además, `depends_on: pater: service_healthy` dejó `nginx` en estado `Created` durante el arranque; un segundo `docker compose up -d nginx` lo inició.
- **Prevención:** `scripts/hurl.sh` ahora hace preflight, levanta el compose si falta PostgreSQL, reintenta nginx y espera a `/healthz` con mensaje claro si sigue sin responder. README documenta la causa del *connection refused*.
- **Inconsistencia de navegación detectada:** con `hx-get` solo se reemplazaba `#main-content`; el `<title>` y el enlace `is-current` de la cabecera quedaban con la página anterior. Se extrajo `NavLinks` como componente compartido, se le dio `id="site-nav"` al bloque y `HTMXResponse` envía `HTMXChrome` con swaps OOB de `<title>` y navegación — sin JavaScript—, manteniendo cabecera/pie estables.
- **Verificación:** test Go de la respuesta OOB, Hurl con aserciones OOB y comprobación en navegador de título/enlace activo en cada ruta.

---

## Futuro (borrador — anotar al concretar)

- Extracción de **Notification** como módulo de Spiritus (Fase 1) → actualizar `units/04` y `05`.
- Extracción de **Identity** (Fase 2, ADR-104) → Pater deja de emitir JWT; nuevo ADR de revisión.
- Migración de módulo→servicio por regla 2-de-3 (ADR-102) → anotar ADR + mover módulo a `services/`.
- Migración NATS→Kafka si se cumple 2-de-3 (ADR-103) → solo cambia el adaptador de la arista.
