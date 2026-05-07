-- name: UpsertBudget :one
INSERT INTO budgets (workspace_id, year, month, total_limit)
VALUES ($1, $2, $3, $4)
ON CONFLICT (workspace_id, year, month)
DO UPDATE SET total_limit = EXCLUDED.total_limit, updated_at = NOW()
RETURNING *;

-- name: GetBudgetByYearMonth :one
SELECT * FROM budgets
WHERE workspace_id = $1 AND year = $2 AND month = $3;

-- name: GetBudgetByID :one
SELECT * FROM budgets WHERE id = $1;

-- name: ListBudgets :many
SELECT * FROM budgets
WHERE workspace_id = $1
ORDER BY year DESC, month DESC;

-- name: DeleteBudget :exec
DELETE FROM budgets WHERE id = $1;

-- name: UpsertBudgetCategory :exec
INSERT INTO budget_categories (budget_id, category_id, limit_amount)
VALUES ($1, $2, $3)
ON CONFLICT (budget_id, category_id)
DO UPDATE SET limit_amount = EXCLUDED.limit_amount;

-- name: DeleteBudgetCategory :exec
DELETE FROM budget_categories WHERE budget_id = $1 AND category_id = $2;

-- name: ListBudgetCategories :many
SELECT bc.budget_id, bc.category_id, bc.limit_amount, c.name AS category_name, c.icon AS category_icon
FROM budget_categories bc
JOIN categories c ON c.id = bc.category_id
WHERE bc.budget_id = $1;

-- name: GetBudgetCategorySpending :many
SELECT
    bc.category_id,
    bc.limit_amount,
    COALESCE(SUM(t.amount), 0)::float8 AS spent
FROM budget_categories bc
LEFT JOIN transactions t
    ON  t.category_id  = bc.category_id
    AND t.workspace_id = $2
    AND t.type         = 'expense'
    AND EXTRACT(YEAR  FROM t.date)::int = $3::int
    AND EXTRACT(MONTH FROM t.date)::int = $4::int
WHERE bc.budget_id = $1
GROUP BY bc.category_id, bc.limit_amount;
