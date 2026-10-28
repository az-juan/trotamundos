# MODIFIABILITY:
- QA-11: Integración de nuevos medios de transporte futuros prevista para el sistema.
---
# AVAILABILITY:
- QA-8: Accesibilidad global 24/7, admitiendo 5 minutos de caída al mes.
- QA-3: Capacidad de mitigar picos extremos ante eventos masivos.
---
# PERFORMANCE:
- QA-5: Actualizaciones en Trotamundos dentro de los 5 minutos posteriores a su generación en terceros.
- QA-6: Tiempo de respuesta en web: 800 ms.
- QA-7: Primer despliegue de contenido en móvil con un tiempo < 1.4 s.
---
# SCALABILITY:
- QA-1: 2M usuarios activos en la semana.
- QA-2: 15M de cuentas de usuario creadas.
- QA-15: Alta carga de datos analíticos para consultas pesadas (incluye la carga de monetización por insight).
---
# INTEROPERABILITY:
- QA-4: Integración con reservas/viajes en apps de terceros (vía APIs), email para su captura, y redes sociales para compartir viajes.
---
# SECURITY:
- QA-9: Acceso seguro y autorizado a los correos del usuario.
- QA-10: Restringir el compartir itinerario con otros usuarios aplicando permisos de visualización.
---
# USABILITY:
- QA-13: Interfaz simple y sencilla, pero funcional. Incluye geolocalización (GPS) en móvil para mostrar reservas cercanas al viajero y deep-linking a apps de terceros (FR-6). Implementado en Pater (Mobile BFF) consumiendo la posición del dispositivo (permiso explícito, solo en foreground).
- QA-14: Opciones Web y móvil.
---
# TESTABILITY:
- QA-12: Requisito de alta trazabilidad y facilidad para evaluaciones futuras.
