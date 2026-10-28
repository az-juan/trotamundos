## ADR-002: Arquitectura Hexagonal por Servicio

**Status**: Aceptado
**Fecha**: 2026-10-26
**Decidido por**: Equipo de arquitectura

> **Adaptación a Trinidad (ADR-102):** la arquitectura hexagonal se conserva **inalterada**, pero se aplica a nivel de **módulo interno dentro de cada unidad** (no de servicio desplegable). El dominio sigue sin importar infraestructura; la Trinidad solo cambia cómo se agrupan los módulos, no su estructura interna.

### Contexto

Trotamundos es una plataforma de gestión de viajes que integra múltiples fuentes (correos, APIs de aerolíneas, hoteles, rentadoras) y debe evolucionar con la incorporación de nuevos medios de transporte (trenes, autobuses). El dominio central (viaje, reserva, ciclo de vida) es estable, pero los adaptadores externos cambian con frecuencia.

### Decisión

Cada microservicio adopta **arquitectura hexagonal (Ports & Adapters)**:

- **Dominio** (`internal/domain/`): entidades, value objects, reglas de negocio. Sin dependencias externas.
- **Use Cases** (`internal/usecase/`): orquestación de dominio. Depende solo de puertos (interfaces).
- **Puertos de salida** (`internal/ports/`): interfaces Go que definen el contrato (p.ej. `BookingRepository`, `EmailProvider`).
- **Adaptadores de entrada** (`internal/adapters/in/`): HTTP handlers, consumers NATS, CLI.
- **Adaptadores de salida** (`internal/adapters/out/`): PostgreSQL, Redis, NATS, clientes REST.

El dominio **nunca importa** `pgx`, `nats.go`, `net/http` ni ninguna librería de infraestructura.

### Consecuencias

| Positivas | Negativas / Riesgos |
|-----------|-------------------|
| El dominio es testeable sin infraestructura (unit tests sin mocks de DB) | Más archivos y capas que un CRUD directo |
| Cambiar de proveedor (p.ej. Gmail → Outlook) = nuevo adaptador, sin tocar dominio | El equipo debe internalizar el patrón (onboarding 1–2 días) |
| Facilita la extracción futura de servicios (un `git mv` + nuevo `go.mod`) | Riesgo de "over-engineering" en servicios pequeños (mitigado: en Fase 0, Notificaciones y Auth son módulos, no servicios) |
| Trazabilidad total: cada puerto tiene un ADR asociado | |
