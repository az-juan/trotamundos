# Sesión con cookie firmada HttpOnly y preferencias persistidas con sqlc.

GET {{host}}/preferences
HTTP 200
[Asserts]
body contains "Identifíquese"

POST {{host}}/auth/login
[Form]
email: dev@trotamundos.app
password: dev
HTTP 200
[Asserts]
header "Set-Cookie" contains "trotamundos_session="
body contains "Panel de viajes"

GET {{host}}/preferences
HTTP 200
[Asserts]
body contains "Preferencias"
body contains "name=\"currency\""

POST {{host}}/preferences
[Form]
locale: es
currency: ARS
gps_radius_km: 75
preferred_agency_id: agencia-sur
HTTP 200
[Asserts]
body contains "Preferencias guardadas."
body contains "ARS"

POST {{host}}/auth/logout
HTTP 200
[Asserts]
body contains "Sesión cerrada."

GET {{host}}/preferences
HTTP 200
[Asserts]
body contains "Identifíquese"
