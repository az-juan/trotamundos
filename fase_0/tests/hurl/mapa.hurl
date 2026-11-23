# Mapa real generado en servidor (FR-12, ADR-112).

GET {{host}}/map
HTTP 200
[Asserts]
body contains "Mapa de viajes"
body contains "/map/image"
body contains "OpenStreetMap"

GET {{host}}/map/image
HTTP 200
[Asserts]
header "Content-Type" contains "image/png"

GET {{host}}/map?status=active
HTTP 200
[Asserts]
body contains "En curso"
body contains "/map/image?status=active"

GET {{host}}/map/image?status=active
HTTP 200
[Asserts]
header "Content-Type" contains "image/png"
