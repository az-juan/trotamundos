# Criterios de transición — Trinidad

Criterios de transición entre las **4 iteraciones de ADD** (CON-5, SPEC §8) y entre **fases** de
despliegue, para la arquitectura **Trinidad**. Un hito está *completado* cuando se cumplen sus criterios;
entonces se mueve el backlog a *Completely Addressed* (Paso 8 de la SPEC) y se avanza.

## Por iteración ADD (CON-5)

| Iteración | Hito (CON-5) | Criterios de transición (completado) | Estado |
|-----------|--------------|---------------------------------------|--------|
| **I** | Referencia y estilos arquitectónicos | • Diagrama de contexto C4 nivel 1 (README §6) ✔<br>• Estilo macro elegido: monolito modular trinitario (ADR-101, ≥3 opciones) ✔<br>• Límites de unidad por perfil de carga (ADR-102) ✔ | **Completada** |
| **II** | Funcionalidad primaria y tecnología | • Módulos por unidad y responsabilidades (`units/01..04`) ✔<br>• Stack definido y **idéntico** al de `main` (tecnologias.md) ✔<br>• Persistencia: PG compartido con esquemas + ClickHouse (ADR-104) ✔<br>• Contratos formales materializados: OpenAPI 3.1 (`docs/api/openapi.yaml`) + eventos JSON Schema (`docs/events/events.yaml`) ✔<br>• Modelo de datos (ERD + DDL + state machine Booking) ✔ | **Completada** (contratos y modelo de datos materializados) |
| **III** | Trade-offs QA, tácticas | • Triángulo de conectores por naturaleza de interacción (ADR-103) ✔<br>• Resiliencia en aristas (timeouts; breaker/retry en Filius) ✔<br>• CQRS: analítica aislada del OLTP (QA-15) ✔<br>• Correlation-id en las 3 aristas (QA-12) ✔ | **Completada** |
| **IV** | Mejoras y preparación del prototipo | • Despliegue 3 contenedores, Docker Compose, DO, GH Actions (ADR-105) ✔<br>• Pipeline template compartido (build-once, deploy ≠ release) ✔<br>• Prototipo ejecutable en 3 deployables (pendiente de construir, CON-4) | **En curso** — falta el prototipo (CON-4) |

## Por fase de despliegue

| Fase | Hito | Criterios de transición (salida) | Estado |
|------|------|----------------------------------|--------|
| **Fase 0** | MVP en 1 Droplet | • 3 contenedores (`pater`/`filius`/`spiritus`) + NATS + Redis en Docker Compose (ADR-105) ✔<br>• PostgreSQL managed con esquemas segregados (ADR-104) ✔<br>• Nginx + TLS como único borde → Pater ✔<br>• OpenAPI 3.1 + eventos JSON Schema materializados ✔<br>• Modelo de datos (DDL) materializado ✔<br>• Guía de dev local + estrategia de testing + operaciones ✔<br>• Smoke tests + verificación de SLO en CI/CD | **Listo para construir** |
| **Fase 1** | Escalado analítico | • 2.º Droplet para ClickHouse + Metabase (dueño: Spiritus)<br>• Redis/NATS managed; NATS a 3 nodos si el volumen lo exige<br>• Extracción de **Notification** como módulo de Spiritus (anotar ADR) | Pendiente (criterio: volumen de eventos o necesidad de notificación en vivo) |
| **Fase 2** | Orquestación + identidad | • K3s + Terraform + ArgoCD (la Trinidad migra **3 despliegues**, no 7)<br>• Extracción de **Identity** (Pater deja de emitir JWT, ADR-104)<br>• Migración NATS→Kafka si se cumple 2-de-3 (ADR-103)<br>• Control de acceso por rol de DB (mitiga DT-01) | Pendiente (criterio: > 8 devs o escalado independiente que exija orquestación) |

## Disparadores de revisión arquitectónica (volver a ADD)

Volver a una iteración de ADD (no seguir de forma pasiva) cuando:

1. **Se cumpla 2-de-3** para extraer un módulo interno a servicio (ADR-102) → Iteración I/II.
2. **Un ítem de deuda** (deuda-arquitectonica.md) pase a *retirarse* → la iteración que corresponda.
3. **Una frontera falle** en producción (fallo en cascada recurrente, latencia no cumplida) → Iteración III.
4. **Un driver nuevo** con impacto arquitectónico profundo aparezca (ASR, Bass ch.19) → Iteración I.

> La Trinidad es **diseño activo** mientras la partición en 3 unidades es el mayor riesgo; pasa a
> **diseño pasivo** (monitorear, corregir, enseñar) una vez el prototipo valide las fronteras, y se
> reactiva al dispararse alguno de los puntos anteriores (Keeling ch.3).
