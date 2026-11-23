#!/usr/bin/env bash
# Genera vistas templ y ejecuta todos los tests del workspace Trinidad.
set -euo pipefail
cd "$(dirname "$0")/.."
(cd units/pater && /home/azjuan/go/bin/templ generate)
for module in shared units/filius units/spiritus units/pater; do
  echo "=== go test $module ==="
  (cd "$module" && go test ./...)
done
