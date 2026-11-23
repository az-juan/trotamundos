# Todas las páginas SSR del panel web (ADR-111) y sus activos locales.

GET {{host}}/
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
body contains "Panel de viajes"
body contains "Buenos Aires → Madrid"
body contains "site-footer"

GET {{host}}/login
HTTP 200
[Asserts]
body contains "Acceso al panel"
body contains "Cookie firmada"

GET {{host}}/trips/view
HTTP 200
[Asserts]
body contains "Viajes"
body contains "Buenos Aires → Madrid"

GET {{host}}/insights/view
HTTP 200
[Asserts]
body contains "Boletín"

GET {{host}}/mailboxes
HTTP 200
[Asserts]
body contains "Buzones"

GET {{host}}/whitelists
HTTP 200
[Asserts]
body contains "Remitentes"

GET {{host}}/nearby?lat=-34.6037&lon=-58.3816&radius_km=5
HTTP 200
[Asserts]
body contains "Cercanías"

GET {{host}}/calendar
HTTP 200
[Asserts]
body contains "Calendario de viajes"
body contains "calendar-grid"

GET {{host}}/static/css/base.css
HTTP 200
[Asserts]
body contains "--wood-dark"

GET {{host}}/static/assets/htmx.min.js
HTTP 200
[Asserts]
header "Content-Type" contains "javascript"
