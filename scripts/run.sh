#!/usr/bin/env bash
# Levanta los tres deployables de la Trinidad en local (prototipo Fase 0).
set -euo pipefail
cd "$(dirname "$0")/.."

./scripts/gen.sh

export PATER_PG_DSN="${PATER_PG_DSN:-postgres://pater_app:pater_dev@127.0.0.1:55432/trotamundos?sslmode=disable}"
echo "Levantando Filius (:8081), Spiritus (:8082) y Pater (:${PATER_PORT:-8090})…"
(cd units/filius   && go run ./cmd/filius)   &
(cd units/spiritus && go run ./cmd/spiritus) &
(cd units/pater    && PATER_PORT="${PATER_PORT:-8090}" go run ./cmd/pater) &

trap 'kill 0 2>/dev/null || true' EXIT INT TERM
wait
