-- ── workspace_invitations ─────────────────────────────────────────────────────
CREATE TABLE workspace_invitations (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email        TEXT        NOT NULL,
    role         TEXT        NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    token        UUID        NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    status       TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'cancelled')),
    invited_by   UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspace_invitations_token     ON workspace_invitations(token);
CREATE INDEX idx_workspace_invitations_workspace ON workspace_invitations(workspace_id);

-- Prevent duplicate pending invitations for the same email in a workspace
CREATE UNIQUE INDEX workspace_invitations_pending_email
    ON workspace_invitations(workspace_id, email)
    WHERE status = 'pending';
