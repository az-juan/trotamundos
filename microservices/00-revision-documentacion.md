# 00 — Revisión de la documentación previa: inconsistencias y ortografía

Auditoría del diseño existente (`../README.md`, `../drivers/`, `../architecture/`, `../adrs/`).
Se listan **errores ortográficos/typos** y **inconsistencias** con ubicación exacta, severidad y
corrección propuesta. Las correcciones de ortografía son seguras (no cambian decisiones); las
inconsistencias de tipo arquitectónico requieren tu decisión y están marcadas como **[DECISIÓN]**.

---

## ✅ Resoluciones aplicadas (2026-10-27)

Las correcciones ortográficas (sección 1) **ya se aplicaron** sobre los archivos originales. Las
inconsistencias de la sección 2 se resolvieron así:

| Ítem | Decisión | Dónde quedó registrada |
|------|----------|------------------------|
| **A. Monetización** | Es un **requisito funcional principal (FR-9)**: la app se monetiza con analítica agregada/anónima vendida a partners. Se agregó FR-9 y se corrigió "Mone4zación"→"Monetización". | `drivers/requerimientos.md`, `05-analytics-service.md` (§Monetización), `vista_modulos.md`, `vista_despliegue.md` |
| **B. Kafka residual** | Kafka es **tecnología futura condicional**; se reemplazaron los nombres residuales por **NATS** (`*_NATS_C`, y Email Ingest como Producer). | `vista_cyc.md` |
| **C. Nodo "Consumer" que publica** | Email Ingest **publica** `booking.discovered` → renombrado a **NATS Producer**. | `vista_cyc.md` |
| **D. Web BFF salta el Gateway** | **Resuelto (ADR-013)**: web y mobile pasan **ambos por el Gateway**, cada uno con **su propio BFF** (Web BFF HTML, Mobile BFF JSON). Se reestructuraron `vista_despliegue.md` y `vista_modulos.md`. | ADR-013, `06-api-gateway.md`, `07-web-bff.md` (§8 Mobile BFF) |
| **I. Origen de los JWT** | **Resuelto (ADR-014)**: **Fase 0** el Gateway emite+valida **JWT RS256** con JWKS público (sin secreto compartido); **Fase 2** se extrae **Identity** como IdP OIDC. | ADR-014, `06-api-gateway.md` |

> Las inconsistencias **E, F, G, H** (idioma de encabezados QA, numeración de QAs, espaciado de
> unidades, redacción del README) son cosméticas: se aprovecharon al aplicar las correcciones de
> ortografía (p.ej. "800 ms"/"1.4 s" con espacio, "móvil", "PERFORMANCE"). Quedan **J, K, L** como
> preguntas abiertas de requisito (definición de whitelist, "viaje completo", QR/boarding passes).

---

## 1. Errores ortográficos y typos

### `drivers/requerimientos.md`

| Línea | Texto actual | Corrección |
|-------|--------------|-----------|
| 9 (FR-1) | "APIs **estandar**" | "APIs **estándar**" |
| 10 (FR-2) | "capturar **automaticamente** las reservas" | "**automáticamente**" |
| 12 (FR-4) | "una **gestion** manual" | "una **gestión** manual" |
| 13 (FR-5) | "archivarlos **automaticamente once** viaje completo" | "**automáticamente una vez** el viaje esté completo" — *"once" = 11 (número); sobra la palabra* |
| 14 (FR-6) | "redirigir **automaticamente** … **atencion** al cliente" | "**automáticamente** … **atención**" |
| 16 (FR-8) | "a **traves** de redes sociales**./**" | "a **través** de redes sociales." — *borrar la `/` final* |

### `drivers/atributos.md`

| Línea | Texto actual | Corrección |
|-------|--------------|-----------|
| 8 | "# **PERFORMACE**:" | "# **PERFORMANCE**:" — *falta la "N"* |
| 2 (QA-11) | "**Integracion** de nuevos medios" | "**Integración**" |
| 5 (QA-8) | "5 minutos de **caida** al mes" | "**caída**" |
| 19 (QA-4) | "**Integracion** con reservas … (**via** APIs)" | "**Integración** … (**vía** APIs)" |
| 24 (QA-10) | "permisos de **visualizacion**" | "**visualización**" |
| 11, 27, 28 (QA-7/13/14) | "en **mobile**" / "GPS en **mobile**" / "Web y **mobile**" | "en **móvil**" (standardizar; ver inconsistencia **G**) |

### `drivers/concerns.md`

| Línea | Texto actual | Corrección |
|-------|--------------|-----------|
| 12 (Iteration III) | "tactics and **refinament**" | "tactics and **refinement**" |
| 13 (Iteration IV) | "More **desicions** and Improvements" | "More **decisions** and Improvements" |

### `architecture/tecnologias.md`

| Línea | Texto actual | Corrección |
|-------|--------------|-----------|
| 5 | "**Lenguage** Android" | "**Lenguaje** Android" — *falta la "j"* |

### `architecture/vista_modulos.md`

| Línea | Texto actual | Corrección |
|-------|--------------|-----------|
| 39 | "ClickHouse … **Mone4zación**" | "**Monetización**" — *error tipográfico "4" por "i"; ver también inconsistencia **A*** |

### `adrs/adr-010.md`

| Línea | Texto actual | Corrección |
|-------|--------------|-----------|
| 26 | "agregaciones en tiempo de **inserta**, no de **query**" | "en tiempo de **inserción**, no de **consulta**" |

### `README.md`

| Línea | Texto actual | Corrección |
|-------|--------------|-----------|
| 22 | "ADR 001: **Division** en microservicios" | "**División**" (el título del ADR-001 sí tiene tilde) |
| 31 | "ADR 010: **Analitica**" | "**Analítica**" |
| 25 | "**Postgresql** + Redis + ClickHouse" | "**PostgreSQL**" (mayúsculas consistentes con los ADRs) |

### Anglicismos técnicos (no son errores estrictos, pero conviene standardizar)

- `adrs/adr-008.md` L19/L30: "**no matcheado**", "**por call**" → sugerido "sin coincidencia", "por llamada".
- `drivers/atributos.md`: "**mobile**" → "**móvil**" (ver **G**).
- `drivers/atributos.md` L19: "email para **sondeo**" → "email para su **captura/sondeo de bandejas**" (redacción).

---

## 2. Inconsistencias

### A. [DECISIÓN] "Monetización" es una función fantasma
`vista_modulos.md` L39 asigna a ClickHouse el propósito "**Monetización**", pero **monetización no
aparece en ningún FR, QA, constraint ni ADR** (FR-1..FR-8 no la mencionan). Hay que **decidir**:
(a) agregarla como requisito explícito (p.ej. un nuevo FR/QA y un caso de uso en Analytics), o
(b) quitarla del diagrama. *Recomendación: (b) por ahora, para no ampliar el alcance del MVP.*

### B. Nombres "Kafka" residuales en `vista_cyc.md`
Los identificadores internos `EI_KAFKA_C`, `IT_KAFKA_C`, `INT_KAFKA_C`, `AN_KAFKA_C` referencian
**Kafka**, pero el ADR-005 fija **NATS JetStream**. Es un residuo de un diseño previo basado en
Kafka. *Corrección: renombrar a `*_NATS_C` (o `*_NS_C`).* Solo afecta legibilidad (son nombres de
nodo), no el comportamiento.

### C. [DECISIÓN] Nodo "Consumer" que en realidad produce
En `vista_cyc.md`, dentro del subgrafo *Email Ingest*, el arco `PARSE_POOL -->|"publish"| EI_KAFKA_C`
llega a un nodo etiquetado "**NATS Consumer** (booking.discovered)". El servicio Email Ingest
**publica** `booking.discovered`, no lo consume → el nodo debería ser un **Producer/Publisher**.
*Corrección: renombrar a "NATS Producer (booking.discovered)".*

### D. [DECISIÓN] El Web BFF salta el Gateway
`vista_despliegue.md` muestra `WEB --> ITIN` y `WEB --> INT` **directamente**, pero el ADR-006 define
al Gateway como "**punto de entrada único para Web y Mobile**". Hay ambigüedad: ¿el BFF llama a los
servicios directamente (patrón BFF, común y válido) o a través del Gateway? *Recomendación: dejarlo
explícito — el BFF puede llamar directo a servicios de dominio en la red interna (BFF = agregador de
su propio frontend), mientras que móviles y terceros pasan por el Gateway. Documentarlo en ADR-006 o
en `06-api-gateway.md`/`07-web-bff.md` de este paquete (ya queda cubierto acá).*

### E. Mezcla de idiomas en los nombres de atributos de calidad
`drivers/atributos.md` usa encabezados en **inglés** (MODIFIABILITY, AVAILABILITY, PERFORMANCE,
SCALABILITY, INTEROPERABILITY, SECURITY, USABILITY, TESTABILITY) con cuerpo en español, mientras el
resto del proyecto está en español. *Recomendación: unificar. Si se mantienen los términos QA en
inglés (habitual en la industria), está bien; pero elegir un criterio y aplicarlo en todos los docs.*

### F. Numeración de QAs dispersa / no secuencial
Las QAs van de QA-1 a QA-15 pero están distribuidas desordenadamente entre categorías
(QA-15 bajo SCALABILITY; QA-13/QA-14 bajo USABILITY; QA-12 bajo TESTABILITY). No es un error, pero
dificulta la trazabilidad cruzada. *Sugerencia: agregar un índice QA→categoría o reordenar.*

### G. Espaciado de unidades ("800ms" vs "800 ms")
`drivers/atributos.md` QA-6/QA-7 usa "800**ms**" y "**1.4s**" (sin espacio); los ADRs usan "800 **ms**"
y "1.4 **s**" (con espacio, norma SI). *Unificar con espacio.*

### H. [DECISIÓN] Redacción del `README.md` (monolito vs microservicios)
L12: "Consideramos el monolito por una cuestión de performance, **pero … preferimos por adoptar** …
microservicios hexagonales". Dos cosas: (1) "*preferimos por adoptar*" es construcción no estándar →
"*preferimos adoptar*" o "*nos inclinamos por adoptar*"; (2) conviene aclarar que se elige
microservicios **a pesar** de la consideración de performance, **por** los drivers de escalabilidad
(ADR-001). *Cosmético pero afecta la lectura del enfoque.*

### I. [DECISIÓN] Origen de los JWT en Fase 0 (breve brecha arquitectónica)
ADR-001 dice que **Auth** es "middleware del Gateway" en Fase 0 y que **Identity** (OAuth/tokens/
social) se extrae recién en Fase 2; `tecnologias.md` lista OAuth2 para "social, APIs B2B". No queda
especificado **quién emite el JWT** en Fase 0 (antes de existir Identity). *Recomendación: documentar
que en Fase 0 el Gateway emite/valida JWT propios (HS256 con secreto compartido) y que en Fase 2 se
migra a Identity como IdP (OpenID Connect). Cubierto en `08-servicios-futuros.md`.*

### J. Límite de la whitelist vs alcance del escaneo (detalle de requisito)
FR-3 (whitelists de remitentes seguros) y FR-2 (capturar automáticamente) se complementan, pero no se
define si el escaneo **solo** procesa remitentes en whitelist, o si procesa todo y la whitelist
solo **prioriza/confía**. *Recomendación: el parser corre sobre todo el correo; la whitelist decide
confianza/escritura directa vs cuarentena. Especificado en `03-email-ingest-service.md`.*

### K. [DECISIÓN/Open] Archivar "viaje completo" (FR-5) — definición del disparo
FR-5 archiva automáticamente al "viaje completo", pero no define cuándo un viaje está completo
(todos los bookings con estado `Completed`? sin fechas futuras? manual del usuario como fallback?).
*Recomendación: viaje completo = todos sus bookings en `Completed`/`Cancelled` y fecha de fin
pasada; con opción manual. Estado machine en `02-itinerary-service.md`.*

### L. [Open] ¿QR / boarding passes?
Las apps de viaje suelen mostrar pases de embarque QR; ningún FR lo menciona. Fuera de alcance actual
pero **vale la pena registrarlo como pregunta abierta** para iteraciones futuras.

---

## 3. Puntos verificados como consistentes (sin acción)

- **Fases 0/1/2**: coherentes entre ADR-001, ADR-004, ADR-005, ADR-010, ADR-012 y `vista_despliegue.md`
  (ClickHouse + Metabase en Droplet 2 desde Fase 1; NATS 1 nodo → cluster de 3).
- **6 deployables Go** (Gateway, Web BFF, Itinerary, Email Ingest, Integration, Analytics): coherente
  entre ADR-001 y `vista_despliegue.md` (NATS/Redis/Nginx contados como infra, no como deployables).
- **SLA 800 ms / FCP 1.4 s**: consistentes en ADR-006, ADR-009, ADR-011 y vistas.
- **Flujo de eventos** `booking.discovered → Itinerary/Integration/Analytics`: coherente entre
  ADR-005, ADR-008 y `vista_cyc.md` (salvo el detalle del nombre "Consumer" del punto **C**).
- **Stack tecnológico** Go 1.23+, PostgreSQL 16, Redis 7, ClickHouse 24.x, NATS 2.10+: consistente
  entre ADR-003/004/005 y `tecnologias.md`.

> **Resumen**: 24 correcciones de ortografía/typo (sección 1) + 12 hallazgos de consistencia
> (7 requerían decisión, marcados **[DECISIÓN]**; 5 abiertos/cosméticos). **Estado actual: todas las
> correcciones ortográficas se aplicaron y las 5 inconsistencias [DECISIÓN] relevantes (A, B, C, D, I)
> se resolvieron** (ver "Resoluciones aplicadas" arriba). Quedan abiertas J, K, L (requisito) y E/F/G/H
> (cosméticas, parcialmente aprovechadas).
