## ADR-005: NATS JetStream como Broker de Eventos

> ⚠️ **Procedencia:** este ADR describe la propuesta original de microservicios (`main`). La decisión vigente para la Trinidad es ADR-101..105. Se conserva como procedencia y adaptación; no es la decisión activa.


**Status**: Aceptado
**Fecha**: 2026-10-26
**Decidido por**: Equipo de arquitectura

> **Adaptación a Trinidad (ADR-103):** NATS JetStream se **restringe a la única arista Filius↔Spiritus**. Pater, el Gateway y los BFF **no** publican ni consumen eventos (leen por REST o read-model). El catálogo de subjects y consumidores se reduce.

### Contexto

Los servicios se comunican de forma asíncrona para: propagar cambios de reserva, notificar disrupciones, alimentar analítica, y desacoplar el escáner de correo del panel. Se evaluaron Kafka y NATS JetStream.

### Decisión

**NATS JetStream** como broker de eventos.

- 1 nodo en Fase 0 (Docker Compose), 3 nodos en cluster a partir de Fase 1.
- Streams con retención configurable (7 días para dead letters, 30 días para eventos de viaje).
- JSON como formato de payload (no Avro, para simplicidad; el schema se documenta en `fase_0/docs/api/events.yaml`).

### Consecuencias

| Positivas | Negativas / Riesgos |
|-----------|-------------------|
| Es un binario Go (< 15 MB, 256 MB RAM). Mismo runtime que los servicios. | No tiene Kafka Connect (no se necesita en esta etapa) |
| TCO ~88% menor que Kafka (benchmark Synadia) | Retención larga (> 3 meses) no es su fortaleza (mitigado: ClickHouse guarda el histórico) |
| Cold start < 1 seg. Sin JVM, sin GC pauses | Si el equipo de datos exige replay de meses de eventos → migrar a Kafka en Fase 2 (los puertos `EventPublisher`/`EventConsumer` lo permiten sin tocar dominio) |
| Cliente Go oficial (`nats.go`), mantenido por Synadia | Ecosistema de herramientas (GUIs, CLI) menor que Kafka |
| Protocolo binario propio: 1–5 μs de overhead vs 5–50 μs de Kafka | |

### Criterio de migración a Kafka

Migrar solo si se cumplen **2 de 3**:
1. El equipo de datos opera Kafka internamente.
2. Se necesita replay de > 3 meses de eventos para el warehouse.
3. El volumen de eventos supera 100K msg/s sostenido.

