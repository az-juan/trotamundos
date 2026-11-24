#!/usr/bin/env bash
# Levanta los tres deployables de la Trinidad en local (prototipo Fase 0).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE="docker compose -f ${ROOT}/fase_0/deploy/docker-compose.yml"
if ! $COMPOSE ps --services --status running | grep -qx postgres; then
	echo "PostgreSQL no está corriendo: levantando docker compose (postgres)..."
	$COMPOSE up -d postgres >/dev/null
	for _ in $(seq 1 30); do
		if $COMPOSE exec -T postgres psql -h 127.0.0.1 -U pater_app -d trotamundos -tAc "SELECT 1 FROM filius.users LIMIT 1" 2>/dev/null | grep -qx 1; then
			break
		fi
		sleep 1
	done
	echo "PostgreSQL iniciado automáticamente y listo en 127.0.0.1:55432."
fi

./scripts/gen.sh

export PATER_PG_DSN="${PATER_PG_DSN:-postgres://pater_app:pater_dev@127.0.0.1:55432/trotamundos?sslmode=disable}"
echo "Levantando Filius (:8081), Spiritus (:8082) y Pater (:${PATER_PORT:-8090})…"
(cd units/filius   && go run ./cmd/filius)   &
(cd units/spiritus && go run ./cmd/spiritus) &
(cd units/pater    && PATER_PORT="${PATER_PORT:-8090}" go run ./cmd/pater) &

cleanup() {
	trap - EXIT INT TERM
	kill 0 2>/dev/null || true
	wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM
wait -n
