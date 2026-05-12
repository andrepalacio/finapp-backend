-- name: CreateSavingsGoal :one
INSERT INTO savings_goals (workspace_id, name, target_amount, deadline, notes)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSavingsGoalByID :one
SELECT * FROM savings_goals WHERE id = $1;

-- name: ListSavingsGoals :many
SELECT * FROM savings_goals WHERE workspace_id = $1 ORDER BY created_at DESC;

-- name: UpdateSavingsGoal :one
UPDATE savings_goals
SET name          = $2,
    target_amount = $3,
    deadline      = $4,
    notes         = $5,
    updated_at    = NOW()
WHERE id = $1 AND workspace_id = $6
RETURNING *;

-- name: DeleteSavingsGoal :exec
DELETE FROM savings_goals WHERE id = $1 AND workspace_id = $2;

-- name: CreateContribution :one
INSERT INTO savings_contributions (goal_id, amount, contributed_at, notes)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetContribution :one
SELECT * FROM savings_contributions WHERE id = $1;

-- name: ListContributions :many
SELECT * FROM savings_contributions
WHERE goal_id = $1
ORDER BY contributed_at DESC, created_at DESC;

-- name: DeleteContribution :exec
DELETE FROM savings_contributions WHERE id = $1 AND goal_id = $2;

-- name: GetTotalContributed :one
SELECT COALESCE(SUM(amount), 0)::float8 AS total FROM savings_contributions WHERE goal_id = $1;
