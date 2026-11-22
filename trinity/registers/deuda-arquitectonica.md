# Deuda arquitectónica — Trinidad

Catálogo de la deuda arquitectónica deliberada de la Trinidad, según **Bass ch.23** y el **Paso 8** de
la SPEC. Cada ítem registra **principal** (costo de retirarla) e **interés** (costo de carga continuo).
Se retira cuando el interés supera al principal. La deuda **estratégica y trazada** es una herramienta,
no un fallo (Bass ch.23).

| ID | Deuda | Por qué se incurrió | Interés (costo de carga) | Principal (costo de retire) | Retirar cuando | Referencia |
|----|-------|---------------------|--------------------------|------------------------------|----------------|------------|
| **DT-01** | **PostgreSQL compartido** (3 esquemas en 1 instancia) en vez de DB física por unidad | Fase 0: equipo pequeño, presupuesto; evitar 3× costo de DB (ADR-104). En Fase 0 incluye el adaptador analítico `PostgresAnalyticsRepo` en el esquema `spiritus` (ADR-107) | **Interés reducido en Fase 0 (ADR-108):** el riesgo de *content coupling* queda bloqueado por RBAC (`pater_app`, `filius_app`, `spiritus_app`). Solo persiste el destino compartido de fallos/IOPS de la instancia PG | Mover la analítica de `Spiritus` a ClickHouse (Fase 1, ADR-107) y extraer el esquema de una unidad a su propia instancia PG si lo requiere (Fase 2) | Fase 1 para analítica (ClickHouse); Fase 2 para OLTP si una unidad cumple la regla 2-de-3 | ADR-104, ADR-107, ADR-108 |
| **DT-02** | **Monolito distribuido** (3 contenedores que se llaman por red Docker) | Colapsar 7 servicios en 3 unidades por perfil de carga (ADR-101/102) | Si una frontera es incorrecta, el peor de ambos mundos; 3 saltos de red exigen timeouts/degradación | Dividir el módulo interno en servicio propio sin tocar dominio | Se cumplan 2 de 3: perfil de escalado, ciclo de deploy, aislamiento de fallos (ADR-102) | ADR-102 |
| **DT-03** | **Sin self-healing / 1 solo Droplet en Fase 0** (SLO Fase 0 de **99.9%** ≈ 43.2 min/mes vs. **99.99%** ≤ 5 min/mes de `QA-8`) | Fase 0: cero overhead de orquestación y presupuesto ~$80/mes (ADR-105, ADR-108) | Recuperación no automática ante fallos de nodo; brecha aceptada de disponibilidad (99.9% vs 99.99% ideal) durante el MVP | Migrar a K3s multi-nodo + ArgoCD (GitOps) para alcanzar el 99.99% de `QA-8` | > 8 devs o escalado/SLA que exija réplicas y auto-healing multi-nodo | ADR-105, ADR-108 |
| **DT-04** | **JWT emitido por el Gateway de Pater** (verificando credenciales vía REST interno contra `filius.users`, ADR-106) | Fase 0: simpleza; un solo borde sin desplegar un IdP separado (ADR-014, ADR-106) | El Gateway firma tokens y depende de `Filius` durante `POST /auth/login` (la validación posterior es stateless vía JWKS) | Extraer Identity como servicio OIDC; Pater valida sin emitir | Fase 2, con OIDC | ADR-014, ADR-106 |
| **DT-06** | **LLM externo del Asistente Virtual** (Fase 1, ADR-109): dependencia de proveedor, coste por sesión y riesgo de PII | Fase 1: soporte conversacional sin plataforma gestionada en Fase 0 (Regla 3 / YAGNI); se mitiga con puerto `IntentClassifier`, ACL + redacción de PII, fallback determinista y presupuesto vigilado | Coste variable y calidad dependiente del proveedor; se controla con circuit breaker y medición de QA-17 | Sustituir el adaptador por uno propio/local, o retirar la deuda si el coste y la tasa de resolución lo justifican | Si el gasto mensual supera el umbral acordado o QA-17 < 70 % sostenido | ADR-109, ADR-110, `ops/operaciones.md` |
| **DT-05** | **NATS de 1 nodo** (sin cluster HA ni tabla Outbox en SQL) | Fase 0: solo una arista lo usa (`Filius↔Spiritus`) | **Interés reducido en Fase 0 (ADR-108):** volumen persistente `nats_data` (`-sd /data`) + retry en memoria con *Exponential Backoff* tras `COMMIT`. Solo queda riesgo ante caída prolongada del nodo único | NATS a cluster de 3 nodos (y opcionalmente Transactional Outbox si se requiere) | El volumen o la criticidad en Fase 1/2 lo exijan | ADR-103, ADR-108 |

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
- **¿Asistente como servicio propio?** No en Fase 1: `assistant/` vive en Pater (ADR-109). Revisar con la regla 2-de-3 si el perfil de carga o el ciclo de deploy divergen.

> **Revisión:** este catálogo es vivo. Se actualiza al final de cada iteración ADD (Paso 8) y cuando un
> ítem pasa a retirarse (se mueve a `transiciones.md` como criterio activo).
