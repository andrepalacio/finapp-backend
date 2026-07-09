//go:build integration

package repositories

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/andrespalacio/finapp-backend/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	migrateOnce sync.Once
	migrateErr  error
)

func testDatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://finapp:finapp@localhost:5432/finapp_test?sslmode=disable"
}

// setupTestDB runs migrations once per test binary, opens a pool, truncates
// all tables so each test starts from a clean slate, and closes the pool on
// test cleanup.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := testDatabaseURL()
	migrateOnce.Do(func() {
		migrateErr = db.RunMigrations(url)
	})
	if migrateErr != nil {
		t.Fatalf("run migrations: %v", migrateErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect test db (is postgres running? TEST_DATABASE_URL=%s): %v", url, err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test db: %v", err)
	}
	t.Cleanup(pool.Close)

	truncateAll(t, pool)
	return pool
}

func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `TRUNCATE TABLE
		savings_contributions, savings_goals,
		debt_payments, debts,
		budget_categories, budgets,
		transactions, transfers,
		workspace_invitations, workspace_members,
		categories, workspaces, users
		RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
