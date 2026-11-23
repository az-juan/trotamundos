# Trotamundos — Arquitectura «Trinidad»

Diseño de software del sistema **Trotamundos** (panel de gestión de viajes web y móvil). La
arquitectura **elegida** es la **Santa Trinidad**: tres unidades modulares interconectadas en triángulo
— **Pater** (el Rostro), **Filius** (el Núcleo) y **Spiritus** (la Mente) — que comparten una sola
sustancia (base de código, dominio y despliegue) y se diferencian en tres personas (responsabilidades
y perfiles de carga distintos).

> Deriva de la misma especificación rectora (**ADD 3.0 + SDD**) y los mismos drivers que la propuesta
> inicial de microservicios, pero elige un monolito **modular partido en 3 unidades** en lugar de 7
> microservicios, conservando y re-particionando todas las decisiones técnicas (Go, hexagonal,
> PostgreSQL/Redis/ClickHouse, NATS, Chi, templ/htmx, Docker Compose, DigitalOcean, GitHub Actions).
> En Fase 0 la sesión web usa cookie firmada y Pater consulta PostgreSQL con sqlc (ADR-111); JWT/OIDC y el BFF móvil son Fase 1.

## Documentación

Toda la documentación de la arquitectura vive en [`trinity/`](./trinity/):

| Sección | Ubicación | Contenido |
|---------|-----------|-----------|
| **Inicio** | [`trinity/README.md`](./trinity/README.md) | Idea central, el triángulo, encaje en el proceso SDD |
| **Vistas arquitectónicas** | [`trinity/architecture/`](./trinity/architecture/) | Módulos (código) · Componentes y conectores (runtime) · Despliegue |
| **Detalle por unidad** | [`trinity/units/`](./trinity/units/) | Pater · Filius · Spiritus + catálogo de eventos |
| **Contratos y datos** | [`trinity/README.md`](./trinity/README.md) | OpenAPI 3.1 · eventos JSON Schema · modelo de datos · dev · testing · ops |
| **Decisiones (ADR)** | [`trinity/adrs/`](./trinity/adrs/) | ADR-101..110 (Trinidad, Hardening Fase 0 y Asistente Virtual Fase 1) + ADR-001..014 (heredados/adaptados) |
| **Drivers** | [`trinity/drivers/`](./trinity/drivers/) | Requerimientos, atributos de calidad, constraints, concerns |
| **Registro de evolución** | [`trinity/registers/`](./trinity/registers/) | Changelog, deuda arquitectónica, criterios de transición |
| **Especificación rectora** | [`skills/SPEC_ARCHITECTURE.md`](./skills/SPEC_ARCHITECTURE.md) | Marco ADD 3.0 + SDD |

## Por qué Trinidad

Conserva el escalado y aislamiento por perfil de carga de los microservicios (batch CPU vs. per-request)
con **solo 3 deployables** en vez de 7: sin service mesh, sin Kubernetes en Fase 0, un solo Droplet con
Docker Compose. Se respeta la sobriedad operativa exigida por las restricciones del proyecto (equipo
pequeño, poco tiempo, presupuesto limitado) y se preserva la vía de extracción de módulo a servicio.

## Capacidad Fase 1: Asistente Virtual

El **Asistente Virtual** ([ADR-109](./trinity/adrs/adr-109-asistente-virtual.md), [ADR-110](./trinity/adrs/adr-110-derivacion-y-estado-asistente.md)) es soporte de primer nivel para reprogramaciones guiadas, consultas de estado e incidencias menores; los casos complejos se derivan a la **agencia preferida** del usuario. Es un módulo de **Pater** (`assistant/`) con un agregado en **Filius** (`assistance/`): **no añade deployables ni aristas**, y su activación está prevista para **Fase 1** (Fase 0 se mantiene sin chat, según `trinity/drivers/design-purpose.md`).

## Prototipo ejecutable Fase 0 (CON-4)

Ya hay un **MVP corrible y testeable** con los tres deployables de la Trinidad:

| Unidad | Módulo Go | Puerto | Responsabilidad en el prototipo |
|--------|-----------|--------|---------------------------------|
| **Pater** | `units/pater` | `8090` | Web BFF: HTML generado con **templ**, CSS con **Grid + Flexbox** e **HTMX local** (sin JS propio) |
| **Filius** | `units/filius` | `8081` | Núcleo OLTP: viajes, reservas, state machine, webhooks firmados y NATS |
| **Spiritus** | `units/spiritus` | `8082` | Ingesta de correo, eventos y read-model de insights (vía NATS) |

```bash
./scripts/test.sh   # templ generate + go test de los 4 módulos
./scripts/run.sh    # levanta las 3 unidades (Pater en :8090 por defecto)
./scripts/seed.sh   # genera datos de demo aleatorios: destinos de Argentina y el mundo, entidades ficticias
./scripts/hurl.sh   # 39 peticiones E2E con Hurl contra el compose (ver abajo)
```

- Navegador: **http://localhost:8090** · salud: `/healthz` en los tres puertos.
- El primer prototipo usa estética **Buenos Aires 1930** (art déco, boletos de tren, sellos) con la
  paleta de [`images/palette.jpg`](./images/palette.jpg): `#335C67`, `#E09F3E`, `#99A88C`, `#FFF3B0`, `#540B0E`.
- Pater usa **PostgreSQL real con sqlc** (`db/queries/` → `shared/db/`); Filius y Spiritus conservan
  adaptadores en memoria para sus tareas de fondo. Las interfaces hexagonales permiten migrarlos sin tocar dominio. Se registra como
  deuda deliberada `DT-07` en `trinity/registers/deuda-arquitectonica.md`.

## Despliegue con Docker Compose (Fase 0)

```bash
cd fase_0/deploy
docker compose up -d --build
docker compose ps          # 7 servicios: nginx, pater, filius, spiritus, nats, redis, postgres
# Si ./scripts/run.sh dice 'connection refused' en 127.0.0.1:55432, el compose está detenido:
# vuelve a ejecutar docker compose up -d (o deja que ./scripts/hurl.sh lo haga)
```

| Servicio | Host | Uso |
|----------|------|-----|
| Nginx (único borde) | `http://localhost:8088` | HTTPS/TLS en producción; en local enruta a Pater |
| Pater | `http://localhost:8090` | Panel web SSR + HTMX + `/map/image` |
| Filius | `http://localhost:8081` | `/healthz` y `/webhooks/{provider}`; el resto de la API REST es referencia Fase 1 |
| Spiritus | `http://localhost:8082` | `/healthz`; ingesta y analítica por NATS (SSE previsto para Fase 1) |
| PostgreSQL / NATS / Redis | `55432` / `4222` / `6379` | PostgreSQL lo usa Pater vía sqlc; NATS el bus `Filius↔Spiritus`; Redis queda preparado para Fase 1 |



### Mapa y estética del prototipo

- `/map` muestra un **mapa real** generado en servidor con `go-staticmaps` y teselas de OpenStreetMap:
  marcadores por reserva geolocalizada y traza entre ellas, sin JavaScript de mapas ni API key.
- La paleta de `images/palette.jpg` se complementa con **tonos madera/café** (`--wood`, `--wood-dark`,
  `--wood-light`, `--coffee`) en marcos, paneles, boletos y botones: estética de café bonaerense de 1930.

### Sin API JSON (ADR-111)

El prototipo Fase 0 **no expone JSON**: los handlers de Pater consultan PostgreSQL directamente
con **sqlc** (`db/queries/`, `shared/db/`) y responden HTML/fragmentos HTMX. La sesión es una
**cookie firmada HttpOnly**; no hay JWT, JWKS ni `/auth/refresh`. El OpenAPI queda como contrato
de referencia para el BFF móvil de Fase 1.

Los puertos host son configurables (`NGINX_HTTP_PORT`, `PATER_HOST_PORT`, `POSTGRES_HOST_PORT`) para
evitar colisiones con otros proyectos locales. La imagen builder usa **Go 1.25** porque `templ v0.3.1020`
requiere Go ≥ 1.25 (dentro de la restricción documentada «Go 1.23+»).

### Páginas y componentes

- **Páginas** (`units/pater/views/`): Panel, Viajes, Detalle de viaje, Boletín, Buzones, Remitentes,
  Preferencias, Itinerario compartido, Acceso, Cercanías (GPS) y Mapa real.
- **Componentes reutilizables** (`views/components.templ`): `Header`, `Footer`, `Panel`, `PageHeading`,
  `Field`, `SubmitButton`, `Badge`, `EmptyState`, `BookingRows`, `BookingForm`, `TripTicket`, `Stats`,
  `Timeline` y `DataList`. Las páginas los componen; no repiten encabezados, tablas ni formularios.
- **Navegación HTMX**: `hx-get` con `hx-target="#main-content"`, `hx-swap="innerHTML"` y `hx-push-url`,
  sin JavaScript propio (único `<script>`: la librería HTMX servida localmente). Las respuestas incluyen
  swaps OOB de `<title>` y `#site-nav` para que el título y el enlace activo sigan al contenido;
  cabecera y pie se reutilizan sin redibujarse.

![Pater-Filius-Spiritus](./images/Pater-Spiritus-Filius.png)
