# Smoke Fase 0: salud de los tres deployables a través el borde y del borde mismo.
# Uso: hurl --variable host=http://127.0.0.1:8088 --test fase_0/tests/hurl/*.hurl

GET {{host}}/healthz
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"
jsonpath "$.unit" == "pater"

GET http://127.0.0.1:8090/healthz
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"

GET http://127.0.0.1:8081/healthz
HTTP 200
[Asserts]
jsonpath "$.unit" == "filius"

GET http://127.0.0.1:8082/healthz
HTTP 200
[Asserts]
jsonpath "$.unit" == "spiritus"

# La capa JSON de Pater fue eliminada (ADR-111).
GET {{host}}/users/me/preferences
HTTP 404

GET {{host}}/auth/refresh
HTTP 404

# Inicio parcial Fase 1: el Asistente Virtual está activo en /assistant (ADR-109/110/113).
GET {{host}}/assistant
HTTP 200
