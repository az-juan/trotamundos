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

- **Disponibilidad objetivo:** en **Fase 0** (1 Droplet) el SLO alcanzable es **99.9%** (~43.2 min/mes de caída), como trade-off deliberado por presupuesto frente al objetivo ideal del **99.99%** (≤ 5 min/mes de `QA-8`), el cual se alcanza en **Fase 2** con K3s multi-nodo (ADR-108, `DT-03`).
- **Deploy:** < 2 min por unidad; build-once, deploy atómico con `--no-deps`.
- **Sin self-healing automático** en Fase 0 (mitigado: `restart: unless-stopped` + UptimeRobot).
- **Sin HA de NATS** en Fase 0 (1 nodo con volumen persistente `nats_data` y retry en memoria con exponential backoff; suficiente porque solo una arista lo usa, ADR-108).
- **Aislamiento RBAC en PostgreSQL desde Fase 0:** roles `pater_app`, `filius_app` y `spiritus_app` restringidos a su propio esquema (ADR-108, mitiga `DT-01`).

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
- API JSON, BFF móvil, JWT/JWKS y clientes HTTP internos en el prototipo Fase 0: los handlers usan sqlc directo, HTMX y cookie firmada (ADR-111).
- Proveedor de LLM o plataforma de chatbot gestionada en Fase 0: el Asistente Virtual se habilita en **Fase 1** detrás del puerto `IntentClassifier`, con fallback determinista y coste monitorizado (ADR-109).

## 7. Disparadores para revisar estas restricciones

- Equipo crece a > 8 devs → revisar K8s (ADR-105).
- Una unidad exige escalado independiente → regla 2-de-3 (ADR-102).
- Volumen de eventos > 100K msg/s o replay > 3 meses → migrar NATS→Kafka (ADR-103).
- Presupuesto permite segundo Droplet → Fase 1 (ClickHouse + Metabase).
- Validar QA-16/QA-17 con datos reales → activar Asistente Virtual en Fase 1 (ADR-109/110).
