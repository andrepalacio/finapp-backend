# finapp-backend

API REST para FinApp — finanzas personales con soporte multi-usuario y workspaces compartidos.

**Stack:** Go 1.22 + Gin + sqlc + PostgreSQL 16 + Redis 7

---

## Inicio rapido

```bash
# 1. Copiar variables de entorno
cp .env.example .env

# 2. Levantar todos los servicios (backend + postgres + redis) con hot reload
make dev

# 3. Verificar que el backend responde
curl http://localhost:8080/health
```

---

## Comandos

| Comando | Descripcion |
|---|---|
| `make dev` | Levanta backend + PostgreSQL + Redis con hot reload (air) |
| `make test` | Corre todos los tests |
| `make test-cover` | Tests con reporte de cobertura HTML |
| `make build` | Build de imagen Docker de produccion (stage runtime) |
| `make migrate-up` | Aplica migraciones pendientes |
| `make migrate-down` | Revierte la ultima migracion |
| `make sqlc` | Regenera codigo Go desde queries SQL |
| `make lint` | Corre golangci-lint |
| `make swagger` | Regenera spec OpenAPI desde comentarios |

---

## Estructura

```
finapp-backend/
├── cmd/api/          # Punto de entrada (main.go)
├── internal/
│   ├── handlers/     # HTTP handlers (Gin)
│   ├── services/     # Logica de negocio
│   ├── repositories/ # Acceso a datos
│   ├── middleware/   # Auth, logging, CORS
│   └── models/       # Structs de dominio
├── pkg/
│   ├── auth/         # JWT
│   ├── response/     # Helpers de respuesta JSON
│   └── validator/    # Validacion de structs
├── db/
│   ├── migrations/   # SQL migrations (golang-migrate)
│   └── queries/      # SQL queries (sqlc)
└── internal/db/      # Codigo generado por sqlc
```

---

## Architecture Decision Records

### ADR-001 — Dos repositorios separados (no monorepo)

**Fecha:** 2026-04-12
**Estado:** Aceptado

**Contexto:** El proyecto tiene un backend en Go y un frontend en Next.js que se despliegan en servidores distintos.

**Decision:** Mantener dos repositorios independientes (`finapp-backend` / `finapp-frontend`) orquestados localmente con `dev.sh`.

**Consecuencias:** Deploys independientes, historiales de git limpios por dominio. Overhead minimo de coordinacion para un equipo pequeño.

---

### ADR-002 — UUID v4 como identificadores

**Fecha:** 2026-04-12
**Estado:** Aceptado

**Contexto:** Se necesita un tipo de ID para todas las entidades del sistema.

**Decision:** Usar UUID v4 generados en la base de datos (`uuid_generate_v4()`).

**Consecuencias:** IDs no predecibles (seguridad), sin dependencia de secuencias auto-increment, facilita merges entre bases de datos si se necesita en el futuro.

---

### ADR-003 — sqlc para acceso a datos

**Fecha:** 2026-04-12
**Estado:** Aceptado

**Contexto:** Se necesita una capa de acceso a datos en Go con PostgreSQL.

**Decision:** Usar `sqlc` (SQL -> Go codegen) en lugar de un ORM.

**Consecuencias:** Queries SQL explicitas y auditables, codigo Go type-safe generado automaticamente, sin magic de ORM. El schema SQL es la fuente de verdad.

---

### ADR-004 — Paginacion mixta

**Fecha:** 2026-04-12
**Estado:** Aceptado

**Contexto:** Distintos endpoints tienen distintas necesidades de paginacion.

**Decision:** Cursor-based para listados grandes (transacciones, movimientos). Offset para listados pequeños y acotados (workspaces de un usuario, miembros de un workspace).

**Consecuencias:** Mejor rendimiento en cursores para datasets grandes; simplicidad en listados cortos.

---

### ADR-005 — Un solo owner por workspace

**Fecha:** 2026-04-12
**Estado:** Aceptado

**Contexto:** Los workspaces tienen roles: `owner`, `admin`, `member`.

**Decision:** Enforced a nivel de base de datos con un indice unico parcial sobre `workspace_members(workspace_id) WHERE role = 'owner'`.

**Consecuencias:** La constraint no se puede violar aunque haya un bug en la app. Transferir ownership requiere una transaccion que cambia ambos roles atomicamente.

---

## Variables de entorno

Ver `.env.example` para la lista completa documentada.
