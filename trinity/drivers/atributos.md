# Atributos de calidad

## Modificabilidad:
- QA-11: Integración de nuevos medios de transporte futuros prevista para el sistema.
---
## Disponibilidad:
- QA-8: Accesibilidad global 24/7, admitiendo 5 minutos de caída al mes.
- QA-3: Capacidad de mitigar picos extremos ante eventos masivos.
---
## Rendimiento:
- QA-5: Actualizaciones en Trotamundos dentro de los 5 minutos posteriores a su generación en terceros.
- QA-6: Tiempo de respuesta en web: 800 ms.
- QA-7: Primer despliegue de contenido en móvil con un tiempo < 1.4 s.
---
## Escalabilidad:
- QA-1: 2M usuarios activos en la semana.
- QA-2: 15M de cuentas de usuario creadas.
- QA-15: Alta carga de datos analíticos para consultas pesadas (incluye la carga de monetización por insight).
---
## Interoperabilidad:
- QA-4: Integración con reservas/viajes en apps de terceros (vía APIs), email para su captura, y redes sociales para compartir viajes.
---
## Seguridad:
- QA-9: Acceso seguro y autorizado a los correos del usuario.
- QA-10: Restringir el compartir itinerario con otros usuarios aplicando permisos de visualización.
---
## Usabilidad:
- QA-13: Interfaz simple y sencilla, pero funcional. Incluye geolocalización para mostrar reservas cercanas al viajero y deep-linking a apps de terceros (FR-6). En Fase 0 se implementa en la **web** mediante el formulario `/nearby` (coordenadas explícitas) y el **mapa real SSR** `/map` (PNG con `go-staticmaps`/OSM, marcadores y trazas, sin JavaScript; ADR-111). El GPS de móvil vuelve en Fase 1.
- QA-14: Opciones Web y móvil.
---
## Testeabilidad:
- QA-12: Requisito de alta trazabilidad y facilidad para evaluaciones futuras.
---
## Asistencia (Fase 1):
- QA-16: Primera respuesta conversacional del Asistente Virtual en **p95 < 2 s** (web y móvil), sin degradar QA-6 (≤ 800 ms) ni QA-7 (FCP < 1.4 s); respuesta por streaming, sin bloquear el render inicial (ADR-109).
- QA-17: Al menos el **70 % de las intenciones** resueltas por el Asistente Virtual sin intervención de la agencia, medible desde los eventos `assistance.*` (ADR-110).
