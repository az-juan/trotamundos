#!/usr/bin/env bash
# Genera las vistas templ: el código *_templ.go no se versiona.
set -euo pipefail
cd "$(dirname "$0")/.."

TEMPL_VERSION="$(cd units/pater && go list -m -f '{{.Version}}' github.com/a-h/templ)"
TEMPL_BIN="$(command -v templ || true)"
if [ -z "$TEMPL_BIN" ] && command -v go >/dev/null 2>&1; then
	GOBIN_DIR="$(go env GOBIN)"
	GOPATH_DIR="$(go env GOPATH)"
	for candidate in "${GOBIN_DIR:+$GOBIN_DIR/templ}" "${GOPATH_DIR:+$GOPATH_DIR/bin/templ}" "$HOME/go/bin/templ"; do
		if [ -n "$candidate" ] && [ -x "$candidate" ]; then
			TEMPL_BIN="$candidate"
			break
		fi
	done
fi

if [ -n "$TEMPL_BIN" ] && [ "$("$TEMPL_BIN" version 2>/dev/null || true)" = "$TEMPL_VERSION" ]; then
	(cd units/pater && "$TEMPL_BIN" generate)
elif command -v go >/dev/null 2>&1; then
	MODCACHE_PROXY="file://$(go env GOMODCACHE)/cache/download"
	(cd units/pater && GOPROXY="${MODCACHE_PROXY},${GOPROXY:-https://proxy.golang.org,direct}" go run "github.com/a-h/templ/cmd/templ@${TEMPL_VERSION}" generate)
else
	echo "Se necesita Go y templ ${TEMPL_VERSION}: go install github.com/a-h/templ/cmd/templ@${TEMPL_VERSION}" >&2
	exit 1
fi
