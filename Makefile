.PHONY: dev test test-cover build migrate-up migrate-down sqlc lint swagger

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
	swag init -g cmd/api/main.go -o api/swagger
