# Design Purpose

> **Propósito del sistema** (driver de negocio, input a ADD Iteración I).

## 1. Visión

Construir la **próxima generación de un panel de gestión de viajes en línea** que permita a los viajeros
ver todas sus reservas existentes organizadas por viaje, ya sea a través de la web o desde su dispositivo
móvil. El objetivo es ofrecer un **único punto de verdad** para todos los arreglos de viaje (hoteles,
autos, vuelos, trenes, autobuses) en una interfaz simple centrada en el viaje, mitigando la frustración
de rastrear múltiples aplicaciones y bandejas de correo.

## 2. Problema que resuelve

- El viajero recibe confirmaciones dispersas (email, apps de aerolíneas/hoteles/rentadoras) y no tiene
  una vista unificada de su viaje.
- El seguimiento manual es propenso a errores y consume tiempo.
- Los datos de viaje son valiosos para analítica (tendencias, demanda, cancelaciones) y monetización.

## 3. Capacidades requeridas

1. **Captura automática** de reservas desde el correo (FR-2, FR-3).
2. **Agrupación por viaje** con auto-archivado al completar (FR-5).
3. **Gestión manual** de reservas (FR-4).
4. **Integración con terceros** (aerolíneas, hoteles, rentadoras) vía APIs y webhooks (FR-1, FR-4, FR-6).
5. **Deep-linking** a apps de terceros (FR-6).
6. **Compartir itinerario** con permisos de visualización (FR-8, QA-10).
7. **Geolocalización (GPS)** en móvil para mostrar reservas cercanas al viajero (QA-13, FR-6).
8. **Analítica y monetización** de datos agregados/anónimos (FR-7, FR-9).
9. **Acceso seguro** a correos del usuario (QA-9).

## 4. Actores

- **Viajero** (web y móvil) — actor principal.
- **Aplicación externa de viajes** — proveedor de reservas (aerolíneas, hoteles, rentadoras).
- **Agencias de viajes** — partners B2B.
- **Mail provider** (Gmail, IMAP) — fuente de captura.
- **Analistas / Partners** — consumidores de insights (FR-7, FR-9).

## 5. Alcance y límites

- **En scope (Fase 0):** captura por email, agrupación por viaje, gestión manual, integración con
  proveedores, compartición, web + móvil (PWA/native), analítica básica, geolocalización (GPS) en móvil.
- **Fuera de scope (Fase 0):** pagos, reservas nuevas (solo lectura/gestión de existentes), chat,
  notificaciones push nativas (Fase 1), identidad independiente (Fase 2).
- **Plataforma:** web (responsive) + iOS (Swift/SwiftUI) + Android (Kotlin/Compose).

## 6. Métricas de éxito

- El viajero ve todas sus reservas organizadas por viaje en < 800 ms (web) / < 1.4 s (FCP móvil).
- Actualización de terceros reflejada en < 5 min.
- Soportar 2M usuarios activos/semana y 15M cuentas (QA-1, QA-2).
- Mitigar picos masivos sin degradar el núcleo transaccional (QA-3, QA-15).
- Geolocalización disponible con permiso explícito, solo en foreground (QA-13).
