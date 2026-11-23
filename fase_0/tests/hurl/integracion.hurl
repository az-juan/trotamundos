# Webhooks firmados y eventos de dominio (ADR-007/103), sin JSON hacia Pater.

POST {{host}}/webhooks/aerolinea
[Form]
payload: {}
HTTP 401

POST {{host}}/webhooks/aerolinea
X-Signature: sha256={{sig}}
Content-Type: text/plain
{"provider_ref":"PNR-1930","status":"disrupted","message":"Hurl E2E"}
HTTP 202

POST {{host}}/webhooks/aerolinea
X-Signature: sha256={{sig_restore}}
Content-Type: text/plain
{"provider_ref":"PNR-1930","status":"confirmed","message":"Hurl E2E restore"}
HTTP 202

GET {{host}}/healthz
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"
