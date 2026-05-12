-- name: CreateInvitation :one
INSERT INTO workspace_invitations (workspace_id, email, role, invited_by, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetInvitationByToken :one
SELECT * FROM workspace_invitations WHERE token = $1;

-- name: GetInvitationByID :one
SELECT * FROM workspace_invitations WHERE id = $1;

-- name: ListPendingInvitations :many
SELECT * FROM workspace_invitations
WHERE workspace_id = $1 AND status = 'pending'
ORDER BY created_at DESC;

-- name: UpdateInvitationStatus :one
UPDATE workspace_invitations SET status = $2 WHERE id = $1 RETURNING *;
