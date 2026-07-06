-- name: CreateTransaction :one
INSERT INTO transactions (workspace_id, user_id, category_id, type, amount, description, date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: CreateTransferRecord :one
INSERT INTO transfers (from_workspace_id, to_workspace_id, note)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateTransferTransaction :one
INSERT INTO transactions (workspace_id, user_id, transfer_id, type, transfer_direction, amount, description, date)
VALUES ($1, $2, $3, 'transfer', $4, $5, $6, $7)
RETURNING *;

-- name: GetTransactionByID :one
SELECT * FROM transactions WHERE id = $1;

-- name: ListTransactions :many
SELECT * FROM transactions
WHERE workspace_id = $1
  AND (sqlc.narg('date_from')::date IS NULL OR date >= sqlc.narg('date_from')::date)
  AND (sqlc.narg('date_to')::date   IS NULL OR date <= sqlc.narg('date_to')::date)
  AND (sqlc.narg('tx_type')::text   IS NULL OR type = sqlc.narg('tx_type')::text)
  AND (sqlc.narg('category_id')::uuid IS NULL OR category_id = sqlc.narg('category_id')::uuid)
ORDER BY date DESC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountTransactions :one
SELECT COUNT(*) FROM transactions
WHERE workspace_id = $1
  AND (sqlc.narg('date_from')::date IS NULL OR date >= sqlc.narg('date_from')::date)
  AND (sqlc.narg('date_to')::date   IS NULL OR date <= sqlc.narg('date_to')::date)
  AND (sqlc.narg('tx_type')::text   IS NULL OR type = sqlc.narg('tx_type')::text)
  AND (sqlc.narg('category_id')::uuid IS NULL OR category_id = sqlc.narg('category_id')::uuid);

-- name: GetDailySummary :many
SELECT
    date,
    SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END)::float8                                AS total_expense,
    SUM(CASE WHEN type = 'income'  THEN amount ELSE 0 END)::float8                                AS total_income,
    SUM(CASE WHEN type = 'transfer' AND transfer_direction = 'out' THEN amount ELSE 0 END)::float8 AS total_transfer_out,
    SUM(CASE WHEN type = 'transfer' AND transfer_direction = 'in'  THEN amount ELSE 0 END)::float8 AS total_transfer_in,
    COUNT(*)::int                                                                                   AS transaction_count
FROM transactions
WHERE workspace_id = $1
  AND (sqlc.narg('date_from')::date IS NULL OR date >= sqlc.narg('date_from')::date)
  AND (sqlc.narg('date_to')::date   IS NULL OR date <= sqlc.narg('date_to')::date)
GROUP BY date
ORDER BY date DESC
LIMIT $2 OFFSET $3;

-- name: GetMonthSummary :one
SELECT
    COALESCE(SUM(CASE WHEN type = 'income'  THEN amount ELSE 0 END), 0)::float8 AS income_total,
    COUNT(CASE WHEN type = 'income'  THEN 1 END)::int                            AS income_count,
    COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)::float8  AS expense_total,
    COUNT(CASE WHEN type = 'expense' THEN 1 END)::int                            AS expense_count
FROM transactions
WHERE workspace_id = @workspace_id
  AND (@date_from::date IS NULL OR date >= @date_from::date)
  AND (@date_to::date   IS NULL OR date <= @date_to::date);

-- name: ListTransactionsByDateCursor :many
SELECT * FROM transactions
WHERE workspace_id = $1
  AND date = $2
  AND (sqlc.narg('cursor')::timestamptz IS NULL OR created_at < sqlc.narg('cursor')::timestamptz)
ORDER BY created_at DESC
LIMIT $3;

-- name: UpdateTransaction :one
UPDATE transactions
SET category_id = $2, amount = $3, description = $4, date = $5, updated_at = NOW()
WHERE id = $1 AND type != 'transfer'
RETURNING *;

-- name: DeleteTransaction :exec
DELETE FROM transactions WHERE id = $1 AND workspace_id = $2;
