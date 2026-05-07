ALTER TABLE workspaces ADD COLUMN currency TEXT NOT NULL DEFAULT 'COP';

-- ── categories ────────────────────────────────────────────────────────────────
CREATE TABLE categories (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID        REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    icon         TEXT,
    color        TEXT,
    type         TEXT        NOT NULL CHECK (type IN ('expense', 'income', 'both')),
    is_system    BOOLEAN     NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX categories_system_name    ON categories(name)                WHERE workspace_id IS NULL;
CREATE UNIQUE INDEX categories_workspace_name ON categories(workspace_id, name)  WHERE workspace_id IS NOT NULL;
CREATE INDEX        idx_categories_workspace  ON categories(workspace_id)        WHERE workspace_id IS NOT NULL;

INSERT INTO categories (name, icon, type, is_system) VALUES
    ('Alimentacion',    '🍔', 'expense', true),
    ('Transporte',      '🚗', 'expense', true),
    ('Salud',           '🏥', 'expense', true),
    ('Educacion',       '📚', 'both',    true),
    ('Entretenimiento', '🎮', 'expense', true),
    ('Hogar',           '🏠', 'expense', true),
    ('Ropa',            '👕', 'expense', true),
    ('Salario',         '💼', 'income',  true),
    ('Freelance',       '💻', 'income',  true),
    ('Inversiones',     '📈', 'income',  true),
    ('Otros',           '📦', 'both',    true);

-- ── transfers ─────────────────────────────────────────────────────────────────
CREATE TABLE transfers (
    id                UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_workspace_id UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    to_workspace_id   UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    note              TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── transactions ──────────────────────────────────────────────────────────────
CREATE TABLE transactions (
    id                 UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id       UUID          NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id            UUID          NOT NULL REFERENCES users(id)      ON DELETE RESTRICT,
    category_id        UUID          REFERENCES categories(id) ON DELETE SET NULL,
    transfer_id        UUID          REFERENCES transfers(id)  ON DELETE CASCADE,
    type               TEXT          NOT NULL CHECK (type IN ('expense', 'income', 'transfer')),
    transfer_direction TEXT          CHECK (transfer_direction IN ('out', 'in')),
    amount             NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    description        TEXT,
    date               DATE          NOT NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_transfer_direction CHECK (
        (type = 'transfer' AND transfer_direction IN ('out', 'in')) OR
        (type != 'transfer' AND transfer_direction IS NULL)
    )
);

CREATE INDEX idx_transactions_workspace_date    ON transactions(workspace_id, date DESC);
CREATE INDEX idx_transactions_workspace_created ON transactions(workspace_id, date, created_at DESC);
CREATE INDEX idx_transactions_transfer          ON transactions(transfer_id) WHERE transfer_id IS NOT NULL;
CREATE INDEX idx_transactions_category          ON transactions(category_id) WHERE category_id IS NOT NULL;

-- ── budgets ───────────────────────────────────────────────────────────────────
CREATE TABLE budgets (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID          NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    year         SMALLINT      NOT NULL,
    month        SMALLINT      NOT NULL CHECK (month BETWEEN 1 AND 12),
    total_limit  NUMERIC(15,2) NOT NULL CHECK (total_limit > 0),
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, year, month)
);

CREATE INDEX idx_budgets_workspace ON budgets(workspace_id, year DESC, month DESC);

-- ── budget_categories ─────────────────────────────────────────────────────────
CREATE TABLE budget_categories (
    budget_id    UUID          NOT NULL REFERENCES budgets(id)    ON DELETE CASCADE,
    category_id  UUID          NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    limit_amount NUMERIC(15,2) NOT NULL CHECK (limit_amount > 0),
    PRIMARY KEY (budget_id, category_id)
);
