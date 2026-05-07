-- name: CreateCategory :one
INSERT INTO categories (workspace_id, name, icon, color, type)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCategoryByID :one
SELECT * FROM categories WHERE id = $1;

-- name: ListCategoriesForWorkspace :many
SELECT * FROM categories
WHERE workspace_id IS NULL OR workspace_id = $1
ORDER BY is_system DESC, name ASC;

-- name: UpdateCategory :one
UPDATE categories SET name = $2, icon = $3, color = $4, type = $5
WHERE id = $1 AND workspace_id IS NOT NULL
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1 AND workspace_id IS NOT NULL AND workspace_id = $2;
