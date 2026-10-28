# Trotamundos — Arquitectura «Trinidad»

Diseño de software del sistema **Trotamundos** (panel de gestión de viajes web y móvil). La
arquitectura **elegida** es la **Santa Trinidad**: tres unidades modulares interconectadas en triángulo
— **Pater** (el Rostro), **Filius** (el Núcleo) y **Spiritus** (la Mente) — que comparten una sola
sustancia (base de código, dominio y despliegue) y se diferencian en tres personas (responsabilidades
y perfiles de carga distintos).

> Deriva de la misma especificación rectora (**ADD 3.0 + SDD**) y los mismos drivers que la propuesta
> inicial de microservicios, pero elige un monolito **modular partido en 3 unidades** en lugar de 7
> microservicios, conservando y re-particionando todas las decisiones técnicas (Go, hexagonal,
> PostgreSQL/Redis/ClickHouse, NATS, Chi, templ/htmx, JWT RS256, Docker Compose, DigitalOcean, GitHub Actions).

## Documentación

Toda la documentación de la arquitectura vive en [`trinity/`](./trinity/):

| Sección | Ubicación | Contenido |
|---------|-----------|-----------|
| **Inicio** | [`trinity/README.md`](./trinity/README.md) | Idea central, el triángulo, encaje en el proceso SDD |
| **Vistas arquitectónicas** | [`trinity/architecture/`](./trinity/architecture/) | Módulos (código) · Componentes y conectores (runtime) · Despliegue |
| **Detalle por unidad** | [`trinity/units/`](./trinity/units/) | Pater · Filius · Spiritus + catálogo de eventos |
| **Decisiones (ADR)** | [`trinity/adrs/`](./trinity/adrs/) | ADR-101..105 (Trinidad) + ADR-001..014 (heredados/adaptados) |
| **Drivers** | [`trinity/drivers/`](./trinity/drivers/) | Requerimientos, atributos de calidad, constraints, concerns |
| **Registro de evolución** | [`trinity/registres/`](./trinity/registres/) | Changelog, deuda arquitectónica, criterios de transición |
| **Especificación rectora** | [`skills/SPEC_ARCHITECTURE.md`](./skills/SPEC_ARCHITECTURE.md) | Marco ADD 3.0 + SDD |

## Por qué Trinidad

Conserva el escalado y aislamiento por perfil de carga de los microservicios (batch CPU vs. per-request)
con **solo 3 deployables** en vez de 7: sin service mesh, sin Kubernetes en Fase 0, un solo Droplet con
Docker Compose. Se respeta la sobriedad operativa exigida por las restricciones del proyecto (equipo
pequeño, poco tiempo, presupuesto limitado) y se preserva la vía de extracción de módulo a servicio.
