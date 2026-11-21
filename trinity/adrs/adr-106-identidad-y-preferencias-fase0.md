# ADR-106: Propiedad del Modelo de Usuario (`users`), Preferencias y Validación de Credenciales en Fase 0

* **Estado:** Aceptado
* **Fecha:** 2026-11-21
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** IV (refinamiento de contratos, datos y preparación del prototipo)

## Contexto y Planteamiento del Problema

En Fase 0, el módulo `gateway` de la unidad **Pater** emite y valida tokens JWT RS256 (ADR-014 adaptado, deuda trazada como `DT-04` hasta la extracción del servicio `Identity` en Fase 2). Sin embargo, el modelo de datos ([modelo-datos.md](../data/modelo-datos.md)) carecía de la definición formal de la tabla `users` (que debe almacenar tanto las credenciales de acceso como el objeto `preferences JSONB` para idioma/locale, moneda, radio GPS de `QA-13` y preferencias de viaje del usuario), a pesar de que `filius.trips`, `spiritus.mailboxes` y `spiritus.whitelist_senders` referencian `user_id UUID NOT NULL`.

Dado que **ADR-104** prohíbe estrictamente el *content coupling* (ninguna unidad puede leer ni escribir directamente en las tablas del esquema de otra unidad), es necesario decidir en qué esquema vive la entidad `users`, cómo valida **Pater** las credenciales durante `POST /auth/login` sin violar las fronteras de esquema, y cómo se expone el perfil de preferencias del viajero para su consumo transaccional y analítico.

## Drivers de Decisión

* **ADR-104 (Capa de datos compartida con segregación por esquema)** — propiedad estricta por esquema (`pater`, `filius`, `spiritus`).
* **ADR-014 (Autenticación JWT RS256)** — Pater emite el token en Fase 0 y publica el JWKS.
* **QA-9 (Seguridad)** — mínimo privilegio y aislamiento de credenciales.
* **QA-6 (Rendimiento web ≤ 800 ms)** — validación de sesión y lectura de perfil de baja latencia.
* **QA-13 (Usabilidad / GPS / Preferencias)** — persistencia extensible de preferencias del usuario (`preferences JSONB`).
* **Constraints** — equipo pequeño, presupuesto acotado y sobriedad operativa (Regla 3 de la SPEC).

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *Propiedad en `Filius` (`filius.users`) + Verificación vía Contrato REST Interno (`Pater → Filius`)*. **(propuesta)**
* **Opción 2:** *Partición entre Credenciales en `Pater` (`pater.auth_identities`) y Perfil/Preferencias en `Filius` (`filius.user_profiles`)*.
* **Opción 3:** *Propiedad Completa de `users` en `Pater` (`pater.users`)*.
* **Opción 4:** *Cuarto Esquema Dedicado (`identity.users`) en el PostgreSQL Compartido*.

## Evaluación Comparativa de Alternativas

### Opción 1: Propiedad en `Filius` (`filius.users`) + Verificación vía Contrato REST Interno (`Pater → Filius`) — propuesta
* **Ventajas:** Mantiene a **Pater** 100% libre de dominio operacional y persistencia transaccional crítica; permite integridad referencial nativa (`FOREIGN KEY`) entre `filius.users(id)` y `filius.trips(user_id)` dentro del mismo esquema; **Filius** tiene acceso local inmediato a `filius.users.preferences` al orquestar viajes y reservas.
* **Desventajas:** Añade un salto REST en la red Docker interna (`< 1 ms`) durante `POST /auth/login` (`Pater → Filius: POST /internal/auth/verify`). No afecta al resto de requests porque la validación posterior del JWT RS256 es stateless en Pater vía clave pública JWKS.
* **Impacto en Restricciones:** Totalmente alineado con equipo pequeño y presupuesto; respeta ADR-104 y ADR-103 sin introducir nuevos esquemas ni sincronización distribuida.

### Opción 2: Partición entre Credenciales en `Pater` (`pater.auth_identities`) y Perfil/Preferencias en `Filius` (`filius.user_profiles`)
* **Ventajas:** **Pater** resuelve el login en su propio esquema sin depender de **Filius**; al extraer `Identity` en Fase 2 (`DT-04`), solo se migra `pater.auth_identities`.
* **Desventajas:** El alta de un usuario exige coordinar escrituras en dos esquemas (`pater` y `filius`), introduciendo consistencia eventual o llamadas de aprovisionamiento entre unidades desde Fase 0.
* **Impacto en Restricciones:** Viable, pero añade complejidad de sincronización innecesaria en Fase 0.

### Opción 3: Propiedad Completa de `users` en `Pater` (`pater.users`)
* **Ventajas:** Login y lectura de perfil resueltos localmente en **Pater** sin salto de red.
* **Desventajas:** Rompe el principio de que **Pater** es un facade sin dominio operacional ([02-pater-facade.md](../units/02-pater-facade.md)); impide claves foráneas nativas en `filius.trips`; obliga a **Filius** o **Spiritus** a consultar a **Pater** si necesitan leer las preferencias del viajero en procesos asíncronos (invirtiendo la dirección del flujo `Pater → Filius`).
* **Impacto en Restricciones:** Simple en arranque, pero genera alto acoplamiento inverso para la evolución del dominio.

### Opción 4: Cuarto Esquema Dedicado (`identity.users`) en el PostgreSQL Compartido
* **Ventajas:** Aísla las tablas de identidad en un namespace SQL propio de cara a la Fase 2.
* **Desventajas:** Rompe la simetría trinitaria «1 unidad desplegable = 1 esquema» de ADR-104 y mantiene la ambigüedad sobre dónde residen las preferencias de viaje del usuario.
* **Impacto en Restricciones:** Diluye las fronteras de propiedad por unidad en Fase 0.

## Decisión Adoptada y Justificación

Se elige la **Opción 1**. La tabla `filius.users` (`id`, `email`, `password_hash`, `display_name`, `preferences JSONB`, `created_at`, `updated_at`) reside en el esquema `filius` bajo propiedad exclusiva de la unidad **Filius**.

1. **Flujo de Login (`POST /auth/login`):** **Pater** recibe las credenciales y delega su verificación llamando por REST interno a **Filius** (`POST /internal/auth/verify`). Si las credenciales son válidas, **Filius** devuelve el `user_id` y metadatos básicos, y el Gateway de **Pater** firma y emite el JWT RS256 junto con el `refresh_token` en Redis.
2. **Gestión de Preferencias (`GET / PATCH /users/me/preferences`):** Se expone en la API pública de **Pater** y se delega por REST a **Filius**, almacenando un documento `JSONB` extensible (`locale`, `currency`, `gps_radius_km` y preferencias de viaje) indexado mediante `GIN`.

## Consecuencias

* **Impacto Positivo:** Se elimina el vacío documental de la entidad `users`; se preserva la pureza de **Pater** como facade stateless; `filius.trips` cuenta con integridad referencial `FOREIGN KEY` hacia `filius.users`; el campo `preferences JSONB` queda listo para personalización y análisis de preferencias del viajero.
* **Trade-offs y Riesgos Aceptados:** Si **Filius** está caído, no es posible iniciar nuevas sesiones (`POST /auth/login`), aunque las sesiones activas con JWT válido siguen siendo verificadas de forma autónoma por **Pater** para servir vistas cacheadas o el read-model de **Spiritus**.
* **Deuda Arquitectónica / Acciones Futuras:** Cuando se retire `DT-04` en Fase 2 (extracción del servicio `Identity` OIDC), las columnas de credenciales (`password_hash`) migrarán al IdP, conservando `filius.users` como entidad de perfil y preferencias del viajero en el dominio operacional.
