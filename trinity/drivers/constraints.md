# Constraints

> Restricciones del proyecto (equipo, tiempo, presupuesto) que guían todas las decisiones arquitectónicas (Regla 3 de la SPEC: *Just Enough Architecture* + *YAGNI*).

## 1. Equipo

- **Tamaño:** 3–5 desarrolladores full-stack (Go backend + iOS/Android + web).
- **Roles:** sin DevOps dedicado en Fase 0; cada dev asume parte de ops.
- **Experiencia:** Go intermedio; familiaridad con Docker Compose; sin experiencia previa en K8s (por eso K8s se reserva para Fase 2).

## 2. Tiempo

- **Horizonte Fase 0 (MVP):** semanas, no meses.
- **Cadencia:** iteraciones cortas; trunk-based (ramas < 1 día).
- **Prototipo ejecutable** (CON-4) al final de la Iteración IV.

## 3. Presupuesto (Fase 0)

- **Total:** ~$80/mes (Droplet 1 $48 + PostgreSQL managed $15–30 + dominio/TLS).
- **Sin K8s, sin service mesh, sin plataformas gestionadas de apps** en Fase 0.
- **ClickHouse + Metabase** (Droplet 2, $96/mes) solo en Fase 1, cuando el volumen lo exija.

## 4. Operacionales

- **Disponibilidad objetivo:** 99.9% (~2h11m/mes de caída, QA-8).
- **Deploy:** < 2 min por unidad; build-once, deploy atómico con `--no-deps`.
- **Sin self-healing automático** en Fase 0 (mitigado: `restart: unless-stopped` + UptimeRobot).
- **Sin HA de NATS** en Fase 0 (1 nodo; suficiente porque solo una arista lo usa).

## 5. Técnicas

- **Stack fijo:** Go 1.23+, PostgreSQL 16, Redis 7, ClickHouse 24, NATS 2.10, Chi v5, templ, htmx 1.9, Tailwind 3.
- **Sin cambios de lenguaje/framework/almacén** respecto de `main`; la Trinidad solo re-particiona.
- **Monorepo multi-módulo** (`go.work` + 3 `go.mod` + `shared/`).

## 6. Lo que NO haremos en Fase 0 (YAGNI)

- Kubernetes / K3s / ArgoCD / Terraform (Fase 2).
- Service mesh (Istio/Linkerd).
- Descubrimiento de servicios (Consul) — todo resuelve por DNS de Docker.
- Base de datos por unidad (compartida con esquemas; Fase 2 si el escalado lo exige).
- Servicio Identity independiente (JWT emitido por el Gateway de Pater; Fase 2).
- Migración NATS → Kafka (solo si se cumple 2-de-3, ADR-103).
- Extracción de Notification como servicio (Fase 1, como módulo de Spiritus).

## 7. Disparadores para revisar estas restricciones

- Equipo crece a > 8 devs → revisar K8s (ADR-105).
- Una unidad exige escalado independiente → regla 2-de-3 (ADR-102).
- Volumen de eventos > 100K msg/s o replay > 3 meses → migrar NATS→Kafka (ADR-103).
- Presupuesto permite segundo Droplet → Fase 1 (ClickHouse + Metabase).
