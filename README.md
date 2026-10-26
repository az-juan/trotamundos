# TPE - TROTAMUNDOS

A continuación presentamos el diseño/arquitectura de software que llevamos a cabo para la implementación del sistema propuesto *Trotamundos*

## Drivers
+ [Requerimientos funcionales](./drivers/requerimientos.md)
+ [Atributos de calidad](./drivers/atributos.md)
+ [Constraints](./drivers/constraints.md)
+ [Concerns](./drivers/concerns.md)

## Arquitectura
Consideramos el monolito por una cuestión de performance, pero teniendo en cuenta los drivers (como la escalabilidad que exige el sistema), preferimos por adoptar un enfoque arquitectónico de microservicios hexagonales + API gateway + broker. \
El diseño hexagonal se debe a lograr una independencia entre la lógica de cada microservicio y las conexiones que tiene a las demás tecnologías, de esta manera logramos que el sistema sea altamente modificable, permitiendo migrar a otras tecnologías cuando el sistema lo requiera (por ejemplo de NATS a Kafka).

+ [Diagrama de contexto](./architecture/diagrama_de_contexto.jpg)
+ [Lista de tecnologías a usar](./architecture/tecnologias.md)
+ [Vista de módulos](./architecture/vista_modulos.md)
+ [Vista de componentes y conectores](./architecture/vista_cyc.md)
+ [Vista de asignaciones/despliegue](./architecture/vista_despliegue.md)

## ADRs
+ [ADR 001: Division en microservicios + Web + Gateway](./adrs/adr-001)
+ [ADR 002: Arquitectura hexagonal por microservicio](./adrs/adr-002)
+ [ADR 003: Go como lenguaje principal](./adrs/adr-003)
+ [ADR 004: Postgresql + Redis + ClickHouse](./adrs/adr-004)
+ [ADR 005: NATS Jetstream como broker de eventos](./adrs/adr-005)
+ [ADR 006: Chi como API Gateway (DIY)](./adrs/adr-006)
+ [ADR 007: Resiliencia en integraciones externas](./adrs/adr-007)
+ [ADR 008: Procesamiento de correo - IMAP + Gmail API + Parser por reglas](./adrs/adr-008)
+ [ADR 009: Web Server-Rendered con templ + htmx + tailwind](./adrs/adr-009)
+ [ADR 010: Analitica - ClickHouse con Batch Ingest](./adrs/adr-010)
+ [ADR 011: Mobile nativo (Swift + Kotlin)](./adrs/adr-011)
+ [ADR 012: Deploy - Docker Compose + GitHub Actions en DigitalOcean](./adrs/adr-012)
