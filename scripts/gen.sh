#!/usr/bin/env bash
# Genera las vistas templ: el código *_templ.go no se versiona.
set -euo pipefail
cd "$(dirname "$0")/.."

if command -v templ >/dev/null 2>&1; then
	(cd units/pater && templ generate)
elif [ -x /home/azjuan/go/bin/templ ]; then
	(cd units/pater && /home/azjuan/go/bin/templ generate)
else
	echo "Se necesita templ v0.3.1020: go install github.com/a-h/templ/cmd/templ@v0.3.1020" >&2
	exit 1
fi
