## ADR-009: Web Server-Rendered con templ + htmx + Tailwind

> ⚠️ **Procedencia:** este ADR describe la propuesta original de microservicios (`main`). La decisión vigente para la Trinidad es ADR-101..105. Se conserva como procedencia y adaptación; no es la decisión activa.


**Status**: Aceptado
**Fecha**: 2026-10-26
**Decidido por**: Equipo de arquitectura

> **Adaptación a Trinidad (ADR-102):** templ + htmx + Tailwind se conservan como **módulo `web-bff` dentro de la unidad Pater** (render del servidor, baja latencia).

### Contexto

La web debe cumplir: FCP < 1.4 s, SLA de 800 ms, i18n multi-región, y una interfaz rica (timeline de viaje, badges de estado, notificaciones en vivo). Se evaluaron: SPA (React/Vue/Svelte), SSR (Next.js/Nuxt), y server-rendered con fragments.

### Decisión

**Go + templ + htmx + Tailwind CSS**:

- **templ** (`github.com/a-h/templ`): templates compilados a `.go` en build time. Type-safe, sin runtime JS de templating.
- **htmx** (~14 KB JS): interactividad sin SPA (swap de fragmentos, SSE para live updates, polling para notificaciones).
- **Tailwind CSS**: utilidades CSS compiladas a estático con CLI. Sin Node.js en runtime.
- **Go chi** como HTTP server del BFF.

### Consecuencias

| Positivas | Negativas / Riesgos |
|-----------|-------------------|
| FCP < 1.4 s: HTML completo en primera respuesta, sin hydration | No es una SPA: navegación completa = request al servidor (mitigado: htmx swap para transiciones parciales) |
| Sin Node.js en runtime: 1 binario Go sirve todo | Ecosistema de componentes frontend más pobre que React (mitigado: htmx cubre el 90% de interactividad) |
| i18n nativo: `templ` + `go-i18n` o `nicksnyder/go-i18n` | Live updates en tiempo real: htmx SSE (no WebSocket bidireccional) |
| Tailwind: 0 CSS en dev, build a 1 archivo estático | |

### Criterio de migración a SPA

Migrar a una SPA (p.ej. SvelteKit) solo si:
1. Se necesita edición colaborativa en tiempo real (2 usuarios editando el mismo itinerario).
2. El FCP se vuelve insuficiente por complejidad de interacción (no es el caso actual).

