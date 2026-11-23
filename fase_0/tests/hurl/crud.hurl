# Gestión manual de viajes/reservas, Deep-linking y archivado (FR-4, FR-5, FR-6, FR-8).

POST {{host}}/trips
[Form]
name: Viaje Hurl Bariloche
start_date: 2026-12-10
end_date: 2026-12-18
HTTP 200
[Asserts]
body contains "Viaje Hurl Bariloche"

GET {{host}}/trips/view
HTTP 200
[Asserts]
body contains "Viaje Hurl Bariloche"

POST {{host}}/bookings
[Form]
trip_id: {{trip_id}}
type: lodging
provider: Hotel Hurl
provider_ref: {{ref}}
HTTP 200
[Asserts]
body contains "trip-list"

POST {{host}}/trips/{{trip_id}}/share
HTTP 200
[Asserts]
body contains "/shared/"

GET {{host}}/shared/{{share_token}}
HTTP 200
[Asserts]
body contains "Itinerario compartido"

POST {{host}}/trips/{{trip_id}}/archive
HTTP 200
[Asserts]
body contains "Viaje archivado correctamente."
