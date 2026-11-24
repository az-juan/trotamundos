#!/usr/bin/env bash
# Levanta los tres deployables de la Trinidad en local (prototipo Fase 0 + Asistente Fase 1).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE="docker compose -f ${ROOT}/fase_0/deploy/docker-compose.yml"
for svc in postgres nats redis; do
	if ! $COMPOSE ps --services --status running | grep -qx "$svc"; then
		echo "Infraestructura ($svc) no está corriendo: levantando postgres, nats y redis con docker compose..."
		$COMPOSE up -d postgres nats redis >/dev/null
		break
	fi
done
for _ in $(seq 1 30); do
	if $COMPOSE exec -T postgres psql -h 127.0.0.1 -U pater_app -d trotamundos -tAc "SELECT 1 FROM filius.users LIMIT 1" 2>/dev/null | grep -qx 1; then
		break
	fi
	sleep 1
done

# Si quedaron contenedores de las unidades o procesos huérfanos ocupando los puertos, se liberan.
$COMPOSE stop nginx pater filius spiritus >/dev/null 2>&1 || true
PATER_LISTEN_PORT="${PATER_PORT:-8090}"
if command -v fuser >/dev/null 2>&1; then
	for port in 8081 8082 "$PATER_LISTEN_PORT"; do
		fuser -k "${port}/tcp" >/dev/null 2>&1 || true
	done
fi

./scripts/gen.sh

export PATER_PG_DSN="${PATER_PG_DSN:-postgres://pater_app:pater_dev@127.0.0.1:55432/trotamundos?sslmode=disable}"
export FILIUS_URL="${FILIUS_URL:-http://127.0.0.1:8081}"
export SPIRITUS_URL="${SPIRITUS_URL:-http://127.0.0.1:8082}"
export NATS_URL="${NATS_URL:-nats://127.0.0.1:4222}"
export REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:6379}"
echo "Levantando Filius (:8081), Spiritus (:8082) y Pater (:${PATER_LISTEN_PORT})…"
(cd units/filius   && go run ./cmd/filius)   &
(cd units/spiritus && go run ./cmd/spiritus) &
(cd units/pater    && PATER_PORT="${PATER_LISTEN_PORT}" go run ./cmd/pater) &

cleanup() {
	trap - EXIT
	trap '' INT TERM
	kill 0 2>/dev/null || true
	wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM
wait -n
