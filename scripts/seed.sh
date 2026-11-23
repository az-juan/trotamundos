#!/usr/bin/env bash
# Aplica datos de demostración aleatorios a la BD del compose ya levantado.
# Cada ejecución genera viajes, reservas, viajeros y eventos nuevos.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="docker compose -f ${ROOT}/fase_0/deploy/docker-compose.yml"

if ! $COMPOSE ps --services --status running | grep -qx postgres; then
  echo "PostgreSQL no está corriendo: docker compose -f fase_0/deploy/docker-compose.yml up -d postgres" >&2
  exit 1
fi

echo "Generando datos de demostración variados…"
$COMPOSE exec -T postgres psql -U trotamundos -d trotamundos -v ON_ERROR_STOP=1 -f - \
  < "${ROOT}/fase_0/deploy/postgres/seed_variety.sql"
echo "Listo: recargue el panel en http://localhost:8090"
