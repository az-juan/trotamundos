# Pruebas E2E del inicio parcial de Fase 1: Asistente Virtual determinista (ADR-109/110/113).
# Verifica consulta Nivel 1, reprogramación guiada (FR-6) y derivación a agencia preferida (FR-11).

GET {{host}}/assistant
HTTP 200
[Asserts]
body contains "Mostrador de Asistencia"
body contains "Consultas rápidas, cambios y gestiones con su agencia"

# 1. Consulta de estado resuelta en primer nivel (status_query)
POST {{host}}/assistant/messages
Content-Type: application/x-www-form-urlencoded
[FormParams]
message: ¿En qué estado están mis próximos viajes y reservas?
HTTP 200
[Asserts]
body contains "Consulta de itinerario"

# 2. Guía de reprogramación mediante enlace directo al proveedor
POST {{host}}/assistant/messages
Content-Type: application/x-www-form-urlencoded
[FormParams]
message: Necesito cambiar la fecha de mi vuelo a Madrid
HTTP 200
[Asserts]
body contains "Cambio o reprogramación"
body contains "proveedores.trotamundos.app/reprogramar"

# 3. Derivación de gestión compleja a la agencia de viajes preferida
POST {{host}}/assistant/messages
Content-Type: application/x-www-form-urlencoded
[FormParams]
message: Solicito reembolso complejo y reclamo con mi agencia preferida
HTTP 200
[Asserts]
body contains "Derivado a su agencia de viajes"
body contains "Confirmar recepción por la agencia"
