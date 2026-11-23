# syntax=docker/dockerfile:1.7
# =============================================================================
# Trotamundos — Multi-Stage Dockerfile para las 3 unidades de la Trinidad
# Targets: pater (:8080), filius (:8081), spiritus (:8082)
# Deriva de: ADR-003 (Go 1.23 estático), ADR-102 (monorepo go.work), ADR-105
# =============================================================================

FROM golang:1.25-alpine AS templgen
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020

FROM golang:1.25-alpine AS builder
WORKDIR /workspace

RUN apk add --no-cache git ca-certificates tzdata
COPY --from=templgen /go/bin/templ /usr/local/bin/templ

COPY go.work* ./
COPY shared/go.mod ./shared/
COPY units/pater/go.mod ./units/pater/
COPY units/filius/go.mod ./units/filius/
COPY units/spiritus/go.mod ./units/spiritus/
RUN go mod download

COPY shared/ ./shared/
COPY units/ ./units/

# Las vistas _templ.go no se versionan: se generan dentro del build.
RUN cd units/pater && templ generate

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pater ./units/pater/cmd/pater && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/filius ./units/filius/cmd/filius && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/spiritus ./units/spiritus/cmd/spiritus

FROM alpine:3.20 AS runtime-base
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S trotamundos && adduser -S -G trotamundos appuser
USER appuser
WORKDIR /app

FROM runtime-base AS pater
COPY --from=builder /out/pater /app/pater
COPY --from=builder /workspace/units/pater/static /app/static
EXPOSE 8080
ENTRYPOINT ["/app/pater"]

FROM runtime-base AS filius
COPY --from=builder /out/filius /app/filius
EXPOSE 8081
ENTRYPOINT ["/app/filius"]

FROM runtime-base AS spiritus
COPY --from=builder /out/spiritus /app/spiritus
EXPOSE 8082
ENTRYPOINT ["/app/spiritus"]
