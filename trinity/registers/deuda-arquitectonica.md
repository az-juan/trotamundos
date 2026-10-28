# Deuda arquitectónica — Trinidad

Catálogo de la deuda arquitectónica deliberada de la Trinidad, según **Bass ch.23** y el **Paso 8** de
la SPEC. Cada ítem registra **principal** (costo de retirarla) e **interés** (costo de carga continuo).
Se retira cuando el interés supera al principal. La deuda **estratégica y trazada** es una herramienta,
no un fallo (Bass ch.23).

| ID | Deuda | Por qué se incurrió | Interés (costo de carga) | Principal (costo de retire) | Retirar cuando | Referencia |
|----|-------|---------------------|--------------------------|------------------------------|----------------|------------|
| **DT-01** | **PostgreSQL compartido** (3 esquemas en 1 instancia) en vez de DB por unidad | Fase 0: equipo pequeño, presupuesto; evitar 3× costo de DB (ADR-104) | Riesgo de *content coupling* si una unidad escribe en esquema ajeno; una caída de PG afecta a las 3 unidades | Extraer el esquema de una unidad a su propia instancia (los puertos hexagonales lo absorben) | Una unidad merezca escalado/aislamiento de DB propio (regla 2-de-3) | ADR-104 |
| **DT-02** | **Monolito distribuido** (3 contenedores que se llaman por red Docker) | Colapsar 7 servicios en 3 unidades por perfil de carga (ADR-101/102) | Si una frontera es incorrecta, el peor de ambos mundos; 3 saltos de red exigen timeouts/degradación | Dividir el módulo interno en servicio propio sin tocar dominio | Se cumplan 2 de 3: perfil de escalado, ciclo de deploy, aislamiento de fallos (ADR-102) | ADR-102 |
| **DT-03** | **Sin self-healing** (Docker Compose, `restart: unless-stopped`) | Fase 0: cero overhead de orquestación (ADR-105) | Recuperación no automática ante fallos de nodo; escalado manual | Migrar a K3s + ArgoCD (GitOps) | > 8 devs o escalado independiente que exija réplicas/orquestación | ADR-105 |
| **DT-04** | **JWT emitido por el Gateway de Pater** (en vez de servicio Identity) | Fase 0: simpleza; un solo borde (ADR-014) | El Gateway carga responsabilidad de auth; acoplamiento auth→facade | Extraer Identity como servicio; Pater valida sin emitir | Fase 2, con OIDC | ADR-014 (adapta ADR-014) |
| **DT-05** | **NATS de 1 nodo** (sin HA) | Fase 0: solo una arista lo usa (Filius↔Spiritus) | Pérdida de eventos no consumidos ante caída del nodo | NATS a cluster de 3 nodos | El volumen o la criticidad lo exijan | ADR-103 |

## Dudas residuales

- **¿Fronteras correctas?** La mayor incertidumbre del monolito distribuido (DT-02) es si la partición
  por perfil de carga es la frontera correcta. **Mitigación**: mantener los módulos internos hexagonales
  para que la extracción sea un `git mv` + nuevo `go.mod` (ADR-102). Revisar en Fase 1 con datos reales
  de escalado.
- **¿Compartir PG o no?** DT-01 es la desviación deliberada del principio «sin DB compartida» (Newman
  ch.2). **Mitigación**: segregación por esquema + propiedad estricta + (Fase 2) control de acceso por
  rol de DB. Vigilar que ninguna unidad escriba en esquema ajeno (regla + revisión).
- **¿Extracción de Notification/Identity en Fase 1/2?** Confirmar con la regla 2-de-3 antes de extraer;
  no adelantar complejidad (Regla 3 de la SPEC, YAGNI).

> **Revisión:** este catálogo es vivo. Se actualiza al final de cada iteración ADD (Paso 8) y cuando un
> ítem pasa a retirarse (se mueve a `transiciones.md` como criterio activo).
