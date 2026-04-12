CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ── users ─────────────────────────────────────────────────────────────────────
CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    name          TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── workspaces ────────────────────────────────────────────────────────────────
CREATE TABLE workspaces (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       TEXT        NOT NULL,
    owner_id   UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspaces_owner ON workspaces(owner_id);

-- ── workspace_members ─────────────────────────────────────────────────────────
CREATE TABLE workspace_members (
    workspace_id UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES users(id)      ON DELETE CASCADE,
    role         TEXT        NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (workspace_id, user_id)
);

-- Un solo owner por workspace
CREATE UNIQUE INDEX workspace_single_owner
    ON workspace_members (workspace_id)
    WHERE role = 'owner';

CREATE INDEX idx_workspace_members_user ON workspace_members(user_id);
