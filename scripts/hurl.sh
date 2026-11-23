#!/usr/bin/env bash
# Pruebas E2E Fase 0 con Hurl contra un docker compose ya levantado.
# Uso: ./scripts/hurl.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOST="${HURL_HOST:-http://127.0.0.1:8088}"
COMPOSE="docker compose -f ${ROOT}/fase_0/deploy/docker-compose.yml"
HURL="${HURL:-hurl}"

if ! command -v "$HURL" >/dev/null 2>&1; then
  if [ -x /home/azjuan/apps/hurl-8.0.1-x86_64-unknown-linux-gnu/bin/hurl ]; then
    HURL=/home/azjuan/apps/hurl-8.0.1-x86_64-unknown-linux-gnu/bin/hurl
  else
    echo "Hurl no encontrado: instala hurl o define HURL=/ruta/a/hurl" >&2
    exit 1
  fi
fi

# Preflight: si el compose está detenido, el DSN 127.0.0.1:55432 da "connection refused".
if ! $COMPOSE ps --services --status running | grep -qx postgres; then
  echo "PostgreSQL no está corriendo: levantando docker compose..."
  $COMPOSE up -d >/dev/null
  # `depends_on: service_healthy` puede dejar nginx en Created si Pater tarda en estar healthy; se reintenta.
  $COMPOSE up -d nginx >/dev/null
  for _ in $(seq 1 30); do
    if curl -fsS "${HOST}/healthz" >/dev/null 2>&1; then break; fi
    sleep 2
  done
fi
if ! curl -fsS "${HOST}/healthz" >/dev/null 2>&1; then
  echo "El borde ${HOST}/healthz no responde; revisa docker compose ps y logs." >&2
  exit 1
fi

psql() { $COMPOSE exec -T postgres psql -U trotamundos -d trotamundos -tAc "$1"; }
TRIP_ID="$(psql "select id from filius.trips where name = 'Buenos Aires → Madrid' limit 1" | tr -d '\r')"
SHARE_TOKEN="$(psql "select token from filius.itinerary_shares where trip_id = '${TRIP_ID}' limit 1" | tr -d '\r')"
if [ -z "${SHARE_TOKEN}" ]; then
  SHARE_TOKEN="$(curl -fsS -X POST "${HOST}/trips/${TRIP_ID}/share" | grep -oE '/shared/[A-Za-z0-9_-]+' | head -1 | cut -d/ -f3)"
fi

PAYLOAD='{"provider_ref":"PNR-1930","status":"disrupted","message":"Hurl E2E"}'
RESTORE='{"provider_ref":"PNR-1930","status":"confirmed","message":"Hurl E2E restore"}'
SECRET="${WEBHOOK_SECRET:-dev-secret}"
SIG="$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" -hex | awk '{print $2}')"
SIG_RESTORE="$(printf '%s' "$RESTORE" | openssl dgst -sha256 -hmac "$SECRET" -hex | awk '{print $2}')"

COMMON=(
  --variable "host=${HOST}"
  --variable "trip_id=${TRIP_ID}"
  --variable "share_token=${SHARE_TOKEN}"
  --variable "ref=HURL-$(date +%s)"
  --variable "sig=${SIG}"
  --variable "sig_restore=${SIG_RESTORE}"
)

echo "== Hurl Fase 0 =="
"$HURL" "${COMMON[@]}" --test "$ROOT"/fase_0/tests/hurl/*.hurl
echo "Hurl: todos los flujos Fase 0 pasaron."
