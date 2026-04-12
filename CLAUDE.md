# finapp-backend — Contexto Claude Code (Backend Go)

> Claude Code lee este archivo automáticamente cuando se ejecuta dentro de `finapp-backend/`.
> Es más específico que el CLAUDE.md raíz. Ambos aplican.

---

## Propósito de este servicio

API REST en Go que expone todos los endpoints de FinApp. Corre en un contenedor Docker independiente. El frontend (Next.js) consume esta API. No hay renderizado de HTML.

**Puerto:** 8080 (interno) → expuesto según `docker-compose.yml`
**Base URL:** `/api/v1/`

---

## Estructura interna detallada

```
internal/
├── handlers/
│   ├── auth.go              # POST /auth/register, /auth/login, /auth/refresh, /auth/logout
│   ├── user.go              # GET/PUT /user/profile
│   ├── transactions.go      # CRUD /transactions
│   ├── budgets.go           # CRUD /budgets
│   ├── debts.go             # CRUD /debts
│   ├── savings.go           # CRUD /savings-goals
│   ├── workspaces.go        # CRUD /workspaces + invitaciones
│   └── reports.go           # GET /reports/monthly, /reports/by-category
├── services/
│   ├── auth_service.go
│   ├── transaction_service.go
│   ├── budget_service.go
│   ├── debt_service.go
│   ├── savings_service.go
│   └── workspace_service.go
├── repositories/
│   ├── db.go                # Inicialización de conexión pgx
│   └── sqlc/                # Código generado por sqlc (NO editar manualmente)
├── middleware/
│   ├── auth.go              # Validación de JWT en cada request
│   ├── workspace.go         # Verifica que el usuario pertenece al workspace
│   ├── cors.go
│   └── logger.go
└── models/
    ├── user.go
    ├── transaction.go
    ├── budget.go
    ├── debt.go
    ├── savings_goal.go
    └── workspace.go
```

---

## Convenciones Go — Este proyecto

### Handlers (Gin)

```go
// PATRÓN ESTÁNDAR de handler en este proyecto
func (h *TransactionHandler) Create(c *gin.Context) {
    var req CreateTransactionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "invalid_body", err.Error())
        return
    }

    workspaceID := middleware.WorkspaceIDFromContext(c)
    userID := middleware.UserIDFromContext(c)

    tx, err := h.svc.Create(c.Request.Context(), services.CreateTransactionParams{
        WorkspaceID: workspaceID,
        UserID:      userID,
        Amount:      req.Amount,
        CategoryID:  req.CategoryID,
        Description: req.Description,
        Date:        req.Date,
    })
    if err != nil {
        response.HandleError(c, err)
        return
    }

    response.Created(c, tx)
}
```

### Servicios

```go
// Los servicios reciben repositorios por interfaz (testeable)
type TransactionService struct {
    repo TransactionRepository  // interfaz, no implementación concreta
}

// Interfaz definida en el mismo paquete del servicio (patrón Go)
type TransactionRepository interface {
    Create(ctx context.Context, params CreateTransactionParams) (Transaction, error)
    List(ctx context.Context, params ListTransactionsParams) ([]Transaction, error)
    GetByID(ctx context.Context, id uuid.UUID) (Transaction, error)
    Update(ctx context.Context, params UpdateTransactionParams) (Transaction, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### Manejo de errores

```go
// Tipo AppError definido en pkg/apperror/
type AppError struct {
    Code       string // ERROR_CODE en snake_case uppercase
    Message    string // Mensaje para el usuario (nunca detalles internos)
    StatusCode int    // HTTP status
    Err        error  // Error original (para logs, nunca para el usuario)
}

// Ejemplos de uso
var (
    ErrNotFound      = &AppError{Code: "NOT_FOUND", StatusCode: 404}
    ErrUnauthorized  = &AppError{Code: "UNAUTHORIZED", StatusCode: 401}
    ErrForbidden     = &AppError{Code: "FORBIDDEN", StatusCode: 403}
    ErrInvalidInput  = &AppError{Code: "INVALID_INPUT", StatusCode: 400}
)

// NUNCA retornar errores de base de datos directamente al usuario
// NUNCA usar fmt.Errorf("sql: ...") como mensaje al usuario
```

### Queries con sqlc

```sql
-- db/queries/transactions.sql
-- name: CreateTransaction :one
INSERT INTO transactions (id, workspace_id, user_id, amount, category_id, description, date, type, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
RETURNING *;

-- name: ListTransactionsByWorkspace :many
SELECT * FROM transactions
WHERE workspace_id = $1
  AND ($2::date IS NULL OR date >= $2)
  AND ($3::date IS NULL OR date <= $3)
ORDER BY date DESC
LIMIT $4 OFFSET $5;
```

> Después de modificar cualquier `.sql` en `db/queries/`, correr `make sqlc` para regenerar Go.

### Tests

```go
// Patrón table-driven (estándar Go)
func TestTransactionService_Create(t *testing.T) {
    tests := []struct {
        name    string
        params  CreateTransactionParams
        want    Transaction
        wantErr bool
    }{
        {
            name:   "valid transaction",
            params: CreateTransactionParams{Amount: 100.50, ...},
            want:   Transaction{Amount: 100.50, ...},
        },
        {
            name:    "negative amount fails",
            params:  CreateTransactionParams{Amount: -10},
            wantErr: true,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // mock del repositorio con mockery o interfaz manual
            repo := &mockTransactionRepository{}
            svc := NewTransactionService(repo)
            got, err := svc.Create(context.Background(), tt.params)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.want.Amount, got.Amount)
        })
    }
}
```

---

## Reglas que Claude Code debe respetar en este proyecto

1. **No editar archivos en `internal/repositories/sqlc/`** — son generados por sqlc
2. **No usar `panic` fuera de `main.go`** — retornar error siempre
3. **No usar `interface{}` o `any` en respuestas de API** — tipar todo explícitamente
4. **No hacer queries SQL directas fuera de repositories** — toda lógica de BD va en repositorios
5. **No exponer detalles de infraestructura en respuestas HTTP** — nunca el mensaje de error de PostgreSQL/Redis
6. **Siempre usar `context.Context`** como primer parámetro en funciones que hacen I/O
7. **Siempre cerrar `rows.Close()` y `defer` resources** — usar `defer` correctamente
8. **No usar variables globales mutables** — pasar dependencias por inyección en constructores

---

## Dependencias principales

```go
// go.mod — dependencias esperadas
github.com/gin-gonic/gin          // HTTP framework
github.com/golang-jwt/jwt/v5      // JWT
github.com/google/uuid            // UUID v4
github.com/jackc/pgx/v5          // Driver PostgreSQL
github.com/redis/go-redis/v9      // Cliente Redis
github.com/sqlc-dev/sqlc          // Generador de código SQL→Go
github.com/golang-migrate/migrate/v4 // Migraciones
github.com/stretchr/testify       // Testing
go.uber.org/zap                   // Logging estructurado
```

---

## Variables de entorno requeridas

```env
# Database
DATABASE_URL=postgres://finapp:finapp@postgres:5432/finapp?sslmode=disable

# Redis
REDIS_URL=redis://redis:6379

# JWT
JWT_SECRET=...          # mínimo 32 caracteres
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h # 7 días

# App
PORT=8080
ENV=development         # development | production
LOG_LEVEL=debug         # debug | info | warn | error
```

---

## Checklist antes de marcar una tarea como completa

- [ ] Handler implementado con validación de input
- [ ] Servicio con lógica de negocio separada
- [ ] Repository con query sqlc
- [ ] Tests del servicio (table-driven, con mock del repo)
- [ ] Test de integración del handler (al menos happy path y error case)
- [ ] Endpoint documentado en api-spec.yaml
- [ ] `make lint` sin errores
