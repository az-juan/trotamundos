## ADR-012: Deploy — Docker Compose + GitHub Actions en DigitalOcean

> ⚠️ **Procedencia:** este ADR describe la propuesta original de microservicios (`main`). La decisión vigente para la Trinidad es ADR-101..105. Se conserva como procedencia y adaptación; no es la decisión activa.


**Status**: Aceptado
**Fecha**: 2026-10-26
**Decidido por**: Equipo de arquitectura

> **Adaptación a Trinidad (ADR-105):** se conserva **Docker Compose + GitHub Actions + DigitalOcean**, pero con **3 imágenes/contenedores** (`pater`/`filius`/`spiritus`) en vez de 7. Deploy atómico por unidad con `--no-deps`.

### Contexto

La startup está en Fase 0 (MVP, equipo de 3–5 devs). No hay necesidad de K8s, multi-región, ni auto-scaling programático. El objetivo es minimizar costo (monetario y de tiempo) mientras se valida el producto.

### Decisión

**Fase 0**:
- **1 Droplet DO** de 8 GB RAM / 4 vCPU ($48/mes) para servicios Go + NATS + Redis.
- **Managed PostgreSQL** DO ($15–$30/mes).
- **Docker Compose** para orquestar containers locales.
- **GitHub Actions** como CI/CD: `push` a `main` → build → test → SSH al Droplet → `docker compose up -d`.
- **Nginx** como reverse proxy + TLS (Certbot / Let's Encrypt).

**Fase 1**: 2 Droplets (blue-green manual) + Managed Redis + ClickHouse en segundo Droplet.

**Fase 2**: K3s (2–3 nodos DO) + Terraform + ArgoCD. Solo cuando se cumplan los criterios de ADR-002 (migración a Kafka) o el equipo supere 10 devs.

### Consecuencias

| Positivas | Negativas / Riesgos |
|-----------|-------------------|
| Costo total Fase 0: ~$80/mes | Sin self-healing (mitigado: `systemd` restart + alertas en UptimeRobot) |
| Deploy en < 2 min (build + push + compose up) | Drift: sin declaratividad (mitigado: `docker-compose.yml` es la fuente de verdad) |
| Sin K8s: cero overhead de orquestación, debugging trivial | Escalado manual (mitigado: agregar Droplet en 5 min desde panel DO) |
| GitHub Actions: reproducible, con tests, sin SSH manual | |

### Criterio de migración a K8s

Migrar cuando se cumplan **2 de 3**:
1. Un servicio necesita escalar de forma independiente (métricas: p99 > 800 ms con réplicas actuales).
2. El equipo supera 8 devs y la coordinación de deploys se vuelve cuello de botella.
3. Se requiere multi-región por compliance.

