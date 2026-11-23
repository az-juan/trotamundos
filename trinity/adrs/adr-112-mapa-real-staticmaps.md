# ADR-112: Mapa real renderizado en servidor con go-staticmaps

* **Estado:** Aceptado (supera la opción SVG de la primera iteración del prototipo)
* **Fecha:** 2026-11-23
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** IV (extensión del prototipo, CON-4)

## Contexto y planteamiento del problema

El usuario pidió visualizar los viajes en un mapa **real** con estética de cafetería porteña de 1930,
sin introducir JavaScript propio ni una API de mapas de pago (ADR-111, CON-3, CON-4).
`filius.bookings.details` ya guarda `lat`/`lon` (QA-13) y `ListGeoPoints` los expone vía sqlc.

| ID | Restricción | Implicación |
|----|-------------|-------------|
| C1 | ADR-111: sin JS propio | Descarta Leaflet/Mapbox GL |
| C2 | Presupuesto ~$80/mes sin plataformas gestionadas | Sin API key ni proveedor de mapas de pago |
| C3 | QA-6/QA-7: primera respuesta útil | La imagen se genera en servidor y se cachea |
| C4 | Paleta 1930 + tonos madera | El tinte se aplica con CSS (`sepia` + marco `wood`), no con tiles modificados |

## Drivers de decisión

* **FR-12** — visualización cartográfica de reservas geolocalizadas.
* **QA-13** — usar coordenadas existentes, sin GPS automático en web.
* **QA-9** — las coordenadas viajan al navegador como imagen, no como telemetría a un SDK third-party.

## Opciones consideradas y evaluación

| Opción | Promueve | Inhibe | Resultado |
|--------|----------|--------|-----------|
| 1. Leaflet + OSM en cliente | interactividad, zoom | JavaScript, CDN, contradice C1 | Descartada |
| 2. Teselas OSM como `<img>` + CSS | sin JS | dependencia de red por imagen, sin trazas | Descartada |
| 3. SVG dibujado a mano en templ | cero dependencias, estilo controlado | **no es un mapa real**: no calles ni barrios | Intentada y sustituida |
| 4. `go-staticmaps` + teselas OSM, PNG en servidor (elegida) | mapa real, sin JS, sin API key, trazas y marcadores | dependencia de red al renderizar, política de uso de OSM | **Aceptada** |

La opción 4 satisface los ASR críticos (C1–C4). `NewTileProviderOpenStreetMaps()` no requiere clave;
`carto` fue probado y devuelve marcas de agua «API KEY REQUIRED», por lo que se descarta.

## Decisión

Pater genera el PNG con `internal/maps.Renderer` usando `github.com/flopp/go-staticmaps` y
`github.com/golang/geo/s2`:

* marcadores burdeos por reserva (`Provider · ProviderRef`),
* traza dorada entre reservas del mismo conjunto,
* ajuste de límites/zoom cuando hay varias coordenadas,
* caché de teselas en `os.TempDir()/trotamundos-tiles` y caché de imagen en memoria (TTL 15 min),
* User-Agent identificable, en cumplimiento de la política de uso de teselas OSM.

El navegador recibe `/map/image` (`Content-Type: image/png`) dentro de un marco madera con filtro CSS
`sepia(.32) saturate(.86) contrast(1.06)` y atribución visible a OpenStreetMap contributors (ODbL).

## Consecuencias

* **Positivas:** mapa real sin JS ni coste; consistencia con ADR-111; estética 1930 conservada; QA-13 verificable.
* **Negativas:** el primer render necesita red hacia el servidor de teselas; si falla, el handler responde 502
  y la página mantiene el marco con el texto alternativo. OSM exige caché y User-Agent respetuosos.
* **Deuda:** no hay reintento ni proveedor alternativo; se registrará si el volumen de render crece.

## Verificación

Prueba de handler sin JS adicional; contenedor con 7 servicios *healthy*; `/map/image` devuelve PNG real
con calles y barrios de Buenos Aires; coordenadas de reservas sobre la imagen; captura revisada visualmente.
