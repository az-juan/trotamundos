# ADR-112: Mapa real renderizado en servidor con go-staticmaps

* **Estado:** Aceptado (supera la opción SVG de la primera iteración del prototipo)
* **Fecha:** 2026-11-23
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** IV (extensión del prototipo, CON-4)

## Contexto y Planteamiento del Problema

Se requiere permitir al viajero visualizar sus viajes y reservas geolocalizadas en un **mapa real** con
estética de cafetería porteña de 1930, sin introducir JavaScript de mapas en el cliente ni contratar
una API de cartografía de pago (ADR-111, CON-3, CON-4). La tabla `filius.bookings.details` ya almacena
`lat` y `lon` (QA-13) y la consulta `ListGeoPoints` los expone mediante `sqlc`.

**Auditoría de consistencia previa (SPEC-001, Regla 2 — Paso 0):**

| ID | Restricción / Decisión vigente | Implicación |
|----|-------------------------------|-------------|
| C1 | ADR-111: sin JavaScript propio en el cliente | Descarta librerías cliente como Leaflet o Mapbox GL JS |
| C2 | `constraints.md`: presupuesto ~$80/mes, sin plataformas SaaS de pago | Descarta proveedores que exijan API key comercial |
| C3 | QA-6 / QA-7: primera respuesta rápida (≤ 800 ms) | La imagen cartográfica debe generarse en servidor y cachearse en memoria/disco |
| C4 | Estética Buenos Aires 1930 + tonos madera | El tinte sepia y el marco de madera se aplican por CSS sobre el PNG servido |

## Drivers de Decisión

* **FR-12** — visualización cartográfica de reservas geolocalizadas y rutas entre destinos.
* **QA-13** — uso de coordenadas de reservas existentes sin requerir rastreo GPS continuo en navegador.
* **QA-9** — privacidad: las coordenadas se procesan en el servidor y viajan al navegador como imagen PNG, sin cargar SDKs de terceros en el cliente.
* **Regla 3 de la SPEC** — sobriedad operativa y presupuesto acotado.

## Opciones Consideradas (Mínimo 3 Alternativas Razonables)

* **Opción 1:** *Leaflet / Mapbox GL JS en el navegador* consumiendo teselas desde el cliente.
* **Opción 2:** *Mosaico de teselas OpenStreetMap como etiquetas `<img>` estáticas + CSS* sin librería de renderizado en servidor.
* **Opción 3:** *Mapa vectorial SVG dibujado a mano en plantillas `templ`* sin cartografía real.
* **Opción 4:** *Renderizado en servidor con `go-staticmaps` + teselas OpenStreetMap cacheadas (`image/png`)* **(propuesta)**.

## Evaluación Comparativa de Alternativas

### Opción 1: Leaflet / Mapbox GL JS en el navegador
* **Ventajas:** zoom y paneo interactivos en cliente; ecosistema conocido.
* **Desventajas:** requiere JavaScript adicional en el navegador y expone la IP del viajero a CDNs externos en cada carga.
* **Impacto en Restricciones:** **contradice** ADR-111 (C1, cero JS propio) y debilita QA-9. Descartada.

### Opción 2: Mosaico de teselas OpenStreetMap como `<img>` estáticas + CSS
* **Ventajas:** no requiere JavaScript en el navegador.
* **Desventajas:** no permite trazar polilíneas geodésicas entre reservas de un mismo viaje ni ajustar dinámicamente el bounding-box y nivel de zoom según la dispersión de puntos.
* **Impacto en Restricciones:** insuficiente para satisfacer `FR-12` (rutas y marcadores por estado). Descartada.

### Opción 3: Mapa vectorial SVG dibujado a mano en `templ`
* **Ventajas:** cero dependencias externas y control estético total.
* **Desventajas:** no representa un mapa real (carece de calles, barrios, costas y escala geográfica mundial cuando se cargan destinos fuera de Buenos Aires).
* **Impacto en Restricciones:** compatible con presupuesto, pero funcionalmente insuficiente para `FR-12`. Sustituida por la Opción 4.

### Opción 4: Renderizado en servidor con `go-staticmaps` + teselas OSM cacheadas — propuesta
* **Ventajas:** genera un mapa cartográfico real en formato PNG (`880×480`) en el servidor con marcadores coloreados por estado (`upcoming`, `active`, `archived`) y trazas entre reservas del mismo viaje; requiere cero JavaScript en cliente y cero API keys comerciales; cachea teselas en disco (`os.TempDir()/trotamundos-tiles`) e imágenes compuestas en memoria (TTL 15 min).
* **Desventajas:** el primer render de una región nueva requiere acceso HTTP saliente desde el servidor hacia el servidor de teselas de OpenStreetMap.
* **Impacto en Restricciones:** **óptimo** — cumple C1–C4, `FR-12`, `QA-9` y `QA-13`.

## Decisión Adoptada y Justificación

Se elige la **Opción 4** porque es la única alternativa que ofrece cartografía real con rutas y marcadores
por estado respetando simultáneamente la prohibición de JavaScript propio de [ADR-111](./adr-111-sin-json-sqlc-htmx.md)
y el presupuesto de ~$80/mes sin API keys comerciales.

Pater implementa `internal/maps.Renderer` con `github.com/flopp/go-staticmaps` y `github.com/golang/geo/s2`:
* Marcadores diferenciados por estado del viaje (`Próximos` en bronce, `En curso` en pizarra, `Archivados` en burdeos).
* Traza dorada conectando las reservas geolocalizadas de cada viaje.
* Cálculo automático de centro y zoom a partir de las coordenadas filtradas.
* Caché de teselas en disco y caché del PNG resultante en memoria por 15 minutos, con `User-Agent` identificable conforme a la política de OpenStreetMap.
* El endpoint `/map/image?status=…` sirve `Content-Type: image/png` y la vista `/map` permite filtrar por estado mediante chips HTMX.

## Consecuencias

* **Impacto Positivo:** mapa real en servidor sin JS ni costos de licenciamiento; coherencia total con ADR-111 y la estética Buenos Aires 1930; privacidad del cliente preservada (`QA-9`).
* **Trade-offs y Riesgos Aceptados:** dependencia de red en servidor durante el primer render de teselas no cacheadas (si el proveedor de teselas no responde, el handler devuelve `502` sin tumbar la página `/map`).
* **Deuda Arquitectónica / Acciones Futuras:** si el tráfico de renderizado cartográfico crece en Fase 2, evaluar un servidor de teselas propio o caché persistente en volumen Docker.
