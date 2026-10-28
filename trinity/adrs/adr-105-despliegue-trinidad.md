# ADR-105: Despliegue — 3 contenedores, Docker Compose, GitHub Actions, DigitalOcean

* **Estado:** Aceptado (rama `main-trinity`)
* **Fecha:** 2026-10-28
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** IV (preparación del prototipo y decisiones operativas)

## Contexto y Planteamiento del Problema

La Trinidad produce **3 deployables** (Pater, Filius, Spiritus) en lugar de los 7 de `main`. Hay que
decidir la plataforma de despliegue manteniendo la **simplicidad operativa** (ADR-012, constraints:
equipo pequeño, poco tiempo, presupuesto). El requisito es **respetar** Docker Compose + GitHub Actions
+ DigitalOcean; la pregunta es cómo mapear las 3 unidades a esa plataforma sin reintroducir overhead.

## Drivers de Decisión

* **CON-4** — prototipo ejecutable tras la última iteración.
* **QA-8 (disponibilidad 24/7, ≤ 5 min/caída/mes)** — arranque rápido, reinicio automático.
* **QA-5/QA-6/QA-7 (rendimiento)** — deploy rápido (< 2 min) y sin desplome por release.
* **ADR-012 (Docker Compose + GH Actions + DO)** — plataforma heredada, se conserva.
* **Constraints** — minimizar costo monetario y de tiempo.

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *Kubernetes / K3s desde el inicio* — orquestación completa (ArgoCD, Terraform) ya en
  Fase 0.
* **Opción 1b (status quo adaptado):** *Docker Compose en 1 Droplet, 3 contenedores* — Nginx + pater +
  filius + spiritus + NATS + Redis, un `docker-compose.yml`. **(propuesta)**
* **Opción 3:** *Plataforma gestionada (App Platform / FaaS de DO)* — desplegar cada unidad como app
  gestionada o función serverless.
* **Opción 4:** *Máquinas desnudas + systemd* — binarios Go directos en el Droplet, sin contenedores.

## Evaluación Comparativa de Alternativas

### Opción 1: Kubernetes / K3s desde el inicio
* **Ventajas:** self-healing, escalado, GitOps. **Desventajas:** overhead operativo enorme para 3
  unidades y un equipo pequeño (ADR-012: K8s solo cuando > 8 devs o criterios 2-de-3); curva de
  aprendizaje; costo de tiempo. **Restricciones:** viola Regla 3 y constraints.

### Opción 1b: Docker Compose, 1 Droplet, 3 contenedores — propuesta
* **Ventajas:** deploy en < 2 min; debugging trivial; cero overhead de orquestación; `docker-compose.yml`
  como fuente de verdad; **deploy atómico por unidad** con `--no-deps` (conserva independencia de
  deploy a nivel de unidad); 3 imágenes en vez de 7. **Desventajas:** sin self-healing (mitigado:
  `restart: unless-stopped` + UptimeRobot); escalado manual. **Restricciones:** óptimo.

### Opción 3: Plataforma gestionada / FaaS
* **Ventajas:** sin gestión de servidores. **Desventajas:** NATS JetStream y PostgreSQL/ClickHouse
  self-hosted no encajan bien en App Platform; cold starts y límites de FaaS para el escáner batch de
  Spiritus; lock-in. **Restricciones:** complica lo que se quiere simple.

### Opción 4: Máquinas desnudas + systemd
* **Ventajas:** sin capa de contenedores. **Desventajas:** gestión manual de dependencias, versions y
  aislamiento; peor reproducibilidad que contenedores; el artefacto ya es un binario Go empaquetado en
  imagen (build-once). **Restricciones:** no aporta; pierde portabilidad.

## Decisión Adoptada y Justificación

Se elige la **Opción 1b**: **1 Droplet DO (8 GB / 4 vCPU, $48/mes) con Docker Compose** conteniendo
Nginx + **3 contenedores Trinidad** (pater, filius, spiritus) + NATS + Redis; **PostgreSQL managed**
($15–30/mes, esquemas segregados, ADR-104); **GitHub Actions** como CI/CD (`push` a `main` → build +
test + contract tests → 3 imágenes → SSH → `docker compose up -d --no-deps <unidad>` → smoke test). Se
conserva **Nginx + TLS (Certbot)** como único borde expuesto, enrutando a Pater. La Trinidad **simplifica**
ADR-012: de 7 a 3 imágenes/pipelines, mismo stack y mismos principios (build-once, deploy ≠ release,
trunk-based). K3s/ArgoCD se reservan para **Fase 2** (mismos criterios de ADR-012: > 8 devs o escalado
independiente que exija réplicas/orquestación).

## Consecuencias

* **Impacto Positivo:** costo Fase 0 ~$80/mes (igual que `main`); deploy < 2 min; 3 pipelines con
  template compartido; deploy atómico por unidad; sin K8s (debugging trivial).
* **Trade-offs y Riesgos Aceptados:** sin self-healing (mitigado: `restart: unless-stopped` + alertas);
  un Droplet no escala horizontalmente (mitigado: escalado vertical + réplicas de Pater en Fase 1;
  extracción de módulo→servicio si el perfil diverge, ADR-102).
* **Deuda Arquitectónica / Acciones Futuras:** **Fase 1** — segundo Droplet para ClickHouse + Metabase
  (Spiritus) y Redis managed; **Fase 2** — K3s + Terraform + ArgoCD y extracción de Identity (ADR-014).
