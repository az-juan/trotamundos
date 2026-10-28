# ADR-102: Límites de unidad y modularidad interna

* **Estado:** Propuesto (rama `main-trinity`)
* **Fecha:** 2026-10-28
* **Decidido por:** Equipo de arquitectura
* **Iteración ADD:** I–II (refinamiento de elementos y responsabilidades)

## Contexto y Planteamiento del Problema

Una vez elegido el estilo *Trinidad* (ADR-101), queda decidir **cómo se particiona** el código en las
tres unidades y **qué arquitectura interna** mantiene cada módulo. La pregunta es dónde trazar los tres
límites de módulo (`units/pater`, `units/filius`, `units/spiritus`) de modo que: (a) cada unidad tenga
un perfil de carga coherente, (b) la comunicación *dentro* de la unidad sea en proceso (sin red), y (c)
los servicios originales (`main`) se conserven como módulos con vía de extracción.

## Drivers de Decisión

* **QA-11 (modifiability)** — módulos con alta cohesión y bajo acoplamiento (Bass ch.8).
* **QA-6/QA-7 (rendimiento)** — Pater debe mantener ≤ 800 ms / < 1.4 s.
* **QA-1/QA-3 (escalabilidad y picos)** — separar batch (Spiritus) de per-request (Filius/Pater).
* **ADR-002 (hexagonal)** — arquitectura interna por módulo, heredada e inalterada.
* **ADR-013 (BFF por experiencia)** — el principio se conserva, ahora como módulos.

## Opciones Consideradas (≥ 3 alternativas razonables)

* **Opción 1:** *1 servicio = 1 unidad* — mantener los 7 servicios como 7 unidades. **(equivale a
  `main`; descartada en ADR-101)**
* **Opción 2:** *Unidad = bounded context individual* — agrupar por bounded context fino (Itinerary,
  Integration, Email Ingest, Analytics, Gateway, BFFs por separado) → 6–7 unidades.
* **Opción 3:** *Unidad = grupo por perfil de carga y estilo de interacción* — **Pater**
  (Gateway+Web BFF+Mobile BFF), **Filius** (Itinerary+Integration), **Spiritus** (Email Ingest+
  Analytics). **(propuesta)**
* **Opción 4:** *Unidad = capa arquitectónica* — una unidad de presentación, una de lógica, una de
  datos (partición por capa / N-tier).

## Evaluación Comparativa de Alternativas

### Opción 1: 1 servicio = 1 unidad
* **Ventajas:** aislamiento máximo. **Desventajas:** 7 unidades = overhead de microservicios (ya
  descartado). **Restricciones:** viola equipo pequeño / presupuesto.

### Opción 2: Unidad = bounded context individual
* **Ventajas:** fronteras alineadas con el dominio. **Desventajas:** en Fase 0 Itinerary↔Integration
  se llaman constantemente (reserva→detalle del proveedor); separarlos añade un salto de red
  innecesario para un perfil de carga idéntico (per-request/OLTP). **Restricciones:** overhead
  intermedio; no aprovecha la cohesión de perfil.

### Opción 3: Unidad = grupo por perfil de carga e interacción — propuesta
* **Ventajas:** cada unidad tiene **un** perfil de carga (Pater=latencia, Filius=OLTP, Spiritus=batch/
  OLAP) → escalado estricto; comunicación interna en proceso (Itinerary↔Integration, gateway↔BFF,
  email-ingest↔analytics = llamadas a función); 3 deployables; la extracción futura divide el módulo
  cuando el perfil diverja. **Desventajas:** requiere disciplina para no compartir memoria entre
  unidades (solo contratos). **Restricciones:** óptimo (equipo pequeño, tiempo, presupuesto).

### Opción 4: Unidad = capa arquitectónica (N-tier)
* **Ventajas:** separación presentación/lógica/datos clásica. **Desventajas:** **acopla por la capa**
  (todos los módulos de presentación dependen de todos los de lógica); impide el escalado por
  responsabilidad de negocio; los datos no son una «unidad» desplegable con sentido (Bass ch.8 critica
  el acoplamiento vertical por capa). **Restricciones:** peor para QA-1/QA-11.

## Decisión Adoptada y Justificación

Se elige la **Opción 3**. Agrupa por **perfil de carga + estilo de interacción**, que es exactamente
lo que determina si algo *debe* estar en procesos separados (Newman ch.1; Bass ch.9/ch.8): Pater
(per-request, latencia, frente al cliente), Filius (per-request, OLTP, dominio operacional), Spiritus
(batch, CPU/OLAP, async). Dentro de cada unidad, los servicios originales de `main` se conservan como
**módulos hexagonales** (ADR-002 inalterado) que se comunican **en proceso**. Esto minimiza los saltos
de red a solo las 3 aristas del triángulo (ADR-103) y preserva la **vía de extracción**. La Opción 4 se
descarta por acoplamiento vertical; la Opción 2 por salto de red innecesario; la Opción 1 es `main`.

## Estructura de módulos resultante

```text
units/pater/        (go.mod)  gateway · web-bff · mobile-bff   → cmd/pater
units/filius/       (go.mod)  itinerary · integration          → cmd/filius
units/spiritus/     (go.mod)  email-ingest · analytics         → cmd/spiritus
shared/             (go.mod)  event · resilient · otelsetup · config · idgen
```

## Consecuencias

* **Impacto Positivo:** 3 unidades con fronteras estables; dentro de cada una, cero red; cada módulo
  sigue hexagonal y testeable sin infra (ADR-002); `one BFF per experience` (ADR-013) se conserva como
  módulos de Pater.
* **Trade-offs y Riesgos Aceptados:** tentación de compartir memoria/estado entre unidades → se
  prohíbe: lo que cruza unidad pasa por contrato (evento NATS o REST, ADR-103).
* **Deuda Arquitectónica / Acciones Futuras:** **regla de extracción 2-de-3** (ADR-001): separar un
  módulo interno en servicio propio solo si 2 de {perfil de escalado, ciclo de deploy, aislamiento de
  fallos} se cumplen. Revisar en Fase 1.
