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
