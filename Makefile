.PHONY: dev test test-cover test-cover-app test-integration build migrate-up migrate-down sqlc lint swagger

# ── Config ────────────────────────────────────────────────────────────────────
MIGRATIONS_DIR = db/migrations
DB_URL         = $(shell grep ^DATABASE_URL .env 2>/dev/null | cut -d '=' -f2-)

# ── Development ───────────────────────────────────────────────────────────────
dev:
	docker compose up --build

# ── Testing ───────────────────────────────────────────────────────────────────
test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Coverage sin codigo generado (sqlc) ni wiring (cmd/api, db) — refleja solo logica de negocio testeable
test-cover-app:
	go test -coverprofile=coverage.out ./...
	@grep -v -E "/repositories/sqlc/|/cmd/api/|finapp-backend/db/" coverage.out > coverage.app.out
	go tool cover -func=coverage.app.out | tail -1
	go tool cover -html=coverage.app.out -o coverage.app.html
	@echo "Coverage report (app only): coverage.app.html"

# Integration tests for internal/repositories against a real Postgres.
# Needs `docker compose up -d postgres` running and a finapp_test database
# (createdb finapp_test, or reuse TEST_DATABASE_URL to point elsewhere).
test-integration:
	go test -tags=integration ./internal/repositories/... -v

# ── Build ─────────────────────────────────────────────────────────────────────
build:
	docker build --target runtime -t finapp-backend:latest .

# ── Database migrations ───────────────────────────────────────────────────────
migrate-up:
	@test -n "$(DB_URL)" || (echo "ERROR: DATABASE_URL no encontrada en .env" && exit 1)
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down:
	@test -n "$(DB_URL)" || (echo "ERROR: DATABASE_URL no encontrada en .env" && exit 1)
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

# ── Code generation ───────────────────────────────────────────────────────────
sqlc:
	sqlc generate

# ── Lint ──────────────────────────────────────────────────────────────────────
lint:
	golangci-lint run ./...

# ── Swagger / OpenAPI ─────────────────────────────────────────────────────────
swagger:
	$(shell go env GOPATH)/bin/swag init -g cmd/api/main.go -o api/swagger
