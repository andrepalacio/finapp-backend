# Arquitectura — finapp-backend

> Snapshot actualizado 2026-07-09. Actualizar cuando cambie la estructura de capas, servicios o cobertura de tests.

## Visión general

API REST en Go (Gin + sqlc + pgx + Redis), monolito por capas. Sirve la API consumida por `finapp-frontend` (Next.js). Sin renderizado HTML, sin llamadas HTTP salientes a otros servicios (no microservicios).

## Capas

```
handlers/     -> validación de input, extrae contexto (workspace/user), llama service, formatea respuesta
services/     -> lógica de negocio, define interfaces Repository por dominio
repositories/ -> implementación con sqlc, único punto de acceso a PostgreSQL
middleware/   -> auth (JWT), workspace (membership check), cors, logger, ratelimit
models/       -> tipos de dominio compartidos
pkg/          -> auth (JWT), response (formato HTTP)
cmd/api/      -> main.go (bootstrap: config/DB/Redis/DI) + router.go (registro de rutas)
```

Regla de dependencia: `handlers -> services -> repositories`. Los servicios reciben repos por interfaz (inyección por constructor), no por implementación concreta — testeable con mocks. `pkg/validator` existió como wrapper sin uso real (dead code) y fue eliminado.

## Manejo de errores

Tipo único `AppError` (`pkg/apperror`). `HandleError` y `Wrap` son el punto central de traducción error -> respuesta HTTP. Nunca se expone el error crudo de Postgres/Redis al cliente.

## Conversión de tipos pg

Centralizada en `internal/repositories/pgconv.go` (`toPgText`, `toPgDate`, `toPgDatePtr`) — reusada por todos los repositorios, sin duplicación.

## Autenticación

JWT (access + refresh) vía `pkg/auth/jwt`. `AuthMiddleware` puebla el contexto con `userID`; `middleware/workspace.go` (`MemberChecker`) valida pertenencia a workspace en rutas que lo requieren.

## Cobertura de tests

| Paquete | Cobertura | Notas |
|---|---|---|
| `pkg/apperror` | 100% | |
| `pkg/response` | 100% | |
| `pkg/auth` | 88.5% | |
| `internal/middleware` | 100% | incluye `ratelimit.go` vía miniredis |
| `internal/services` | 72.5% | los 9 servicios tienen test unitario, incl. `invitation` |
| `internal/handlers` | 77.7% | los 9 handlers antes sin test (budget/category/debt/savings/transaction/workspace/invitation/import/alert) ahora cubiertos |
| `internal/repositories` | 51.8% con `-tags=integration` | integration tests reales contra Postgres para `workspace`/`debt`/`savings`/`transaction`; `category`/`budget`/`invitation`/`user` repos sin test (CRUD puro, menor prioridad) |

`make test-cover-app` (excluye sqlc generado + wiring de `cmd/api`/`db`) para el número agregado real.

`make test-integration` corre los tests de `internal/repositories` contra Postgres real (`docker compose up -d postgres`, DB `finapp_test` separada de dev). Fuera del suite normal (`go test ./...`), aislado por build tag.

## Puntos fuertes

- Capas respetadas sin fugas (no hay SQL crudo en handlers/services, no hay `interface{}`/`any` en capa de negocio salvo `DBTX` generado).
- Sin variables globales mutables.
- Sin `panic()` fuera de rutas esperadas.
- Error handling y conversión pg centralizados — bajo acoplamiento disperso.
- `cmd/api/main.go` separado de `router.go` (wiring vs rutas) — evita que `main()` crezca sin límite.

## Riesgos conocidos

- `internal/repositories`: `category_repository`, `budget_repository`, `invitation_repository`, `user_repository` sin integration test (CRUD puro sqlc, bajo riesgo pero sin cobertura real).
- Rutas duplicadas `/login` vs `/auth/login` en el grafo de código: verificado como artefacto del indexer, no duplicación real en `router.go`.
