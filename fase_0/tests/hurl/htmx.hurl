# Fragmentos HTMX: sin JSON, sin JavaScript propio (ADR-111).

GET {{host}}/trips
HX-Request: true
HTTP 200
[Asserts]
body contains "trip-list"
body contains "hx-swap-oob=\"true\""
body contains "id=\"filters\""

# Las páginas HTMX sí redibujan navegación y título vía OOB.
GET {{host}}/mailboxes
HX-Request: true
HTTP 200
[Asserts]
body contains "id=\"site-nav\""
body contains "hx-swap-oob=\"true\""

GET {{host}}/insights
HX-Request: true
HTTP 200
[Asserts]
body contains "stat-grid"

GET {{host}}/trips?status=active
HX-Request: true
HTTP 200

POST {{host}}/demo/ingest
HX-Request: true
HTTP 200
[Asserts]
body contains "trip-list"
