-- name: CreateDebt :one
INSERT INTO debts (workspace_id, name, lender, principal, rate, rate_type, installments, first_payment_date, notes, insurance_rate, insurance_type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetDebtByID :one
SELECT * FROM debts WHERE id = $1;

-- name: ListDebts :many
SELECT * FROM debts WHERE workspace_id = $1 ORDER BY created_at DESC;

-- name: UpdateDebt :one
UPDATE debts
SET name               = $2,
    lender             = $3,
    principal          = $4,
    rate               = $5,
    rate_type          = $6,
    installments       = $7,
    first_payment_date = $8,
    notes              = $9,
    insurance_rate     = $10,
    insurance_type     = $11,
    updated_at         = NOW()
WHERE id = $1 AND workspace_id = $12
RETURNING *;

-- name: DeleteDebt :exec
DELETE FROM debts WHERE id = $1 AND workspace_id = $2;

-- name: CreateDebtPayment :one
INSERT INTO debt_payments (debt_id, period, amount, paid_at, notes)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDebtPayment :one
SELECT * FROM debt_payments WHERE id = $1;

-- name: ListDebtPayments :many
SELECT * FROM debt_payments WHERE debt_id = $1 ORDER BY period ASC;

-- name: UpdateDebtPayment :one
UPDATE debt_payments SET amount = $2, paid_at = $3, notes = $4
WHERE id = $1
RETURNING *;

-- name: DeleteDebtPayment :exec
DELETE FROM debt_payments WHERE id = $1 AND debt_id = $2;
