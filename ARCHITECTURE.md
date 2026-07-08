# Arquitectura — finapp-backend

> Snapshot generado por evaluación de arquitectura (2026-07-05). Actualizar cuando cambie la estructura de capas o servicios.

## Visión general

API REST en Go (Gin + sqlc + pgx + Redis), monolito por capas. Sirve la API consumida por `finapp-frontend` (Next.js). Sin renderizado HTML, sin llamadas HTTP salientes a otros servicios (no microservicios).

## Capas

```
handlers/     -> validación de input, extrae contexto (workspace/user), llama service, formatea respuesta
services/     -> lógica de negocio, define interfaces Repository por dominio
repositories/ -> implementación con sqlc, único punto de acceso a PostgreSQL
middleware/   -> auth (JWT), workspace (membership check), cors, logger, ratelimit
models/       -> tipos de dominio compartidos
pkg/          -> auth (JWT), response (formato HTTP), validator
```

Regla de dependencia: `handlers -> services -> repositories`. Los servicios reciben repos por interfaz (inyección por constructor), no por implementación concreta — testeable con mocks.

## Métricas (grafo de código)

- 1427 nodos / 4304 edges
- 66 rutas, 9 servicios, 5 middlewares
- 210 clases/tipos, 514 métodos, 249 funciones

## Manejo de errores

Tipo único `AppError` (`pkg/apperror`). `HandleError` y `Wrap` son el punto central de traducción error -> respuesta HTTP (56 call sites cada uno). Nunca se expone el error crudo de Postgres/Redis al cliente.

## Conversión de tipos pg

Centralizada en `internal/repositories/pgconv.go` (`toPgText`, `toPgDate`, `toPgDatePtr`) — reusada por todos los repositorios, sin duplicación.

## Autenticación

JWT (access + refresh) vía `pkg/auth/jwt`. `AuthMiddleware` puebla el contexto con `userID`; `middleware/workspace.go` (`MemberChecker`) valida pertenencia a workspace en rutas que lo requieren.

## Cobertura de tests

| Servicio | Test unitario |
|---|---|
| auth | sí |
| budget | sí |
| category | sí |
| debt | sí |
| savings | sí |
| transaction | sí |
| user | sí |
| workspace | sí |
| invitation | **no** |

## Puntos fuertes

- Capas respetadas sin fugas (no hay SQL crudo en handlers/services, no hay `interface{}`/`any` en capa de negocio salvo `DBTX` generado).
- Sin variables globales mutables.
- Sin `panic()` fuera de rutas esperadas.
- Error handling y conversión pg centralizados — bajo acoplamiento disperso.

## Riesgos conocidos

Ver propuesta de mejoras: `.claude/proposals/architecture-improvements.md`.
