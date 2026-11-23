# ADR-111: Prototipo Fase 0 sin API JSON — HTMX + sqlc + sesión por cookie

* **Estado:** Aceptado
* **Fecha:** 2026-11-23
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** IV (revisión del prototipo ejecutable, CON-4)

## Contexto y Planteamiento del Problema

El prototipo construido incorporó un API JSON en Pater (mobile BFF), clientes HTTP hacia Filius y
Spiritus, JWT RS256 con JWKS y un bus de eventos mixto. El equipo solicita eliminar esas solicitudes
JSON y usar **directamente la lógica y los handlers con sqlc**, HTMX y cookies, sin JavaScript propio.

**Auditoría previa (SPEC-001 Regla 2 — Paso 0):**

| ID | Decisión vigente | Conflicto | Resolución |
|----|------------------|-----------|------------|
| C1 | ADR-006 / ADR-013 | Gateway con API JSON y BFF móvil | El **BFF JSON se difiere a Fase 1**; en Fase 0 solo se sirve HTML/HTMX. |
| C2 | ADR-014 | JWT RS256 + JWKS | Sustituido en Fase 0 por **cookie firmada HMAC, HttpOnly y SameSite=Lax**; JWT/OIDC vuelve con el canal móvil. |
| C3 | ADR-103 | Pater↔Filius por REST | Pater usa **sqlc directamente** sobre PostgreSQL compartido; NATS se conserva solo para `Filius↔Spiritus`. |
| C4 | ADR-104 / ADR-108 | Escritura solo en el esquema propio | Se amplían los privilegios de `pater_app` sobre tablas/vistas `filius` en el prototipo; el objetivo RBAC estricto se mantiene para Fase 1. |
| C5 | OpenAPI 3.1 | Contrato JSON expuesto | Se conserva como **contrato de referencia para Fase 1**, no como superficie implementada en Fase 0. |
| C6 | QA-14 (web y móvil) | El móvil necesita JSON | El alcance de Fase 0 queda **web-only**; se registra la reducción en `design-purpose.md`. |

## Drivers de Decisión

* **Regla 3 de la SPEC** — equipo pequeño, poco tiempo, presupuesto acotado: menos capas y menos contratos.
* **CON-4** — prototipo desechable: validar la UI y el dominio con la menor infra intermedia posible.
* **QA-6/QA-7** — SSR con HTMX mantiene HTML en la primera respuesta; sin hidratación.
* **ADR-002** — hexagonal: los handlers dependen de un puerto de persistencia implementado por sqlc.
* **QA-9** — sesión en cookie HttpOnly, sin token expuesto a JavaScript.

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *Mantener API JSON + BFF móvil + JWT* — estado actual del prototipo.
* **Opción 2:** *HTMX + sqlc directo + cookie* — los handlers consultan PostgreSQL con sqlc; sin JSON interno. **(propuesta)**
* **Opción 3:** *HTMX + fachada Go en memoria* — quitar JSON pero seguir con repositorios en memoria (sin sqlc).
* **Opción 4:** *Server Components / WebSockets* — render reactivo con runtime adicional.

## Evaluación Comparativa de Alternativas

### Opción 1: API JSON + BFF móvil + JWT
* **Ventajas:** canales web y móvil separados; contratos versionados; escalado independiente.
* **Desventajas:** duplica modelos y validaciones; más capas para un equipo de 3–5 devs; el prototipo depende de 4 saltos de red por pantalla.
* **Impacto en restricciones:** sobre-ingeniería para CON-4.

### Opción 2: HTMX + sqlc directo + cookie — propuesta
* **Ventajas:** un solo flujo de datos (handler → sqlc → PostgreSQL → HTML); cero JS propio; sesiones opacas en cookie; SQL tipado y revisable; permite probar handlers con fakes del puerto.
* **Desventajas:** acopla Pater al esquema `filius` en el prototipo; el móvil requiere reintroducir un BFF en Fase 1.
* **Impacto en Restricciones:** óptimo para el MVP; alinea implementación con el guidance de sqlc directo en handlers.

### Opción 3: HTMX + repositorios en memoria
* **Ventajas:** mismas vistas, sin base de datos.
* **Desventajas:** no valida el modelo de datos ni los índices/RBAC del despliegue real; deuda `DT-07` intacta.
* **Impacto en Restricciones:** insuficiente para cerrar Fase 0.

### Opción 4: Server Components / WebSockets
* **Ventajas:** interactividad rica.
* **Desventajas:** runtime adicional, mayor complejidad y contradice «sin JS» del prototipo.
* **Impacto en Restricciones:** descartada.

## Decisión Adoptada y Justificación

Se elige la **Opción 2**. Los handlers de `units/pater` dejan de consumir JSON de otras unidades y
llaman a un puerto de persistencia implementado con **sqlc/pgx**; las respuestas son fragmentos HTML
intercambiados por HTMX. La autenticación usa una **cookie firmada** con HMAC-SHA256 (`user_id`, `exp`),
sin JWKS. Filius conserva los webhooks y el auto-archivado, y Spiritus el consumo NATS/analítica,
ambos sobre el mismo esquema SQL. El bus JetStream no cambia.

## Consecuencias

* **Impacto positivo:** menos red, menos serialización, un único modelo de datos tipado; navegación y formularios 100 % HTMX; sesión invisible al cliente JS.
* **Trade-offs aceptados:** Pater accede a `filius` en el prototipo (ver C4); sin canal móvil en Fase 0; el OpenAPI queda diferido.
* **Deuda / acciones futuras:** Fase 1 reintroduce BFF JSON + JWT/OIDC y restaura RBAC estricto por esquema; se registran como `DT-08` y `DT-09`.
