# Propuesta de mejoras — arquitectura finapp-backend

**Fecha:** 2026-07-05
**Base:** evaluación de arquitectura vía grafo de código (1427 nodos, 4304 edges)

Ningún hallazgo estructural grave. Arquitectura por capas cumple sus propias reglas (CLAUDE.md). Mejoras son puntuales, no requieren refactor.

## 1. `invitation_service.go` sin test unitario

Único servicio de los 9 sin `*_test.go`. Resto (auth, budget, category, debt, savings, transaction, user, workspace) sí tiene.

**Acción:** crear `internal/services/invitation_service_test.go`, tabla-driven, mock de `InvitationRepository`/`InvitationUserRepository`/`InvitationWorkspaceRepository`, siguiendo el patrón de `workspace_service_test.go`.

## 2. Cobertura de tests despareja entre servicios

Solo 42 edges TESTS totales en el grafo, concentrados en algunos servicios. Verificar manualmente si `debt_service`, `savings_service`, `transaction_service` cubren casos de error (no solo happy path) además de creación.

**Acción:** correr `make test-cover`, revisar % por archivo en `internal/services/`, priorizar servicios con lógica de negocio más compleja (transferencias, cálculo de cuotas de deuda).

## 3. Rutas duplicadas con y sin prefijo `/auth`

Grafo muestra `/login`, `/register`, `/refresh`, `/logout` (sin prefijo) coexistiendo con `/auth/login`, `/auth/register`, `/auth/refresh`, `/auth/logout`.

**Acción:** confirmar si es alias legacy intencional (compat con frontend viejo) o remanente de refactor. Si es remanente, eliminar las rutas sin prefijo — reduce superficie de API y confusión en swagger.

## 4. `main.go` con fan-out alto (38 calls)

Esperado en wiring de dependencias (constructores + registro de rutas). No es lógica de negocio filtrada, pero vale vigilar que no crezca con lógica que debería vivir en services.

**Acción:** ninguna ahora. Revisar si `main.go` supera ~150 líneas o empieza a tener condicionales de negocio — señal de que necesita extraerse a un paquete `bootstrap`.

## No se recomienda

- No dividir en microservicios: monolito por capas es adecuado al tamaño actual (9 servicios, 66 rutas), sin señales de necesidad de escalado independiente por dominio.
- No tocar el mecanismo de error handling centralizado (`HandleError`/`Wrap`, 56 call sites cada uno): está bien diseñado, cambiarlo sería refactor sin beneficio.
