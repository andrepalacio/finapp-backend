-- ── debts ─────────────────────────────────────────────────────────────────────
CREATE TABLE debts (
    id                 UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id       UUID          NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name               TEXT          NOT NULL,
    lender             TEXT,
    principal          NUMERIC(15,2) NOT NULL CHECK (principal > 0),
    rate               NUMERIC(10,8) NOT NULL CHECK (rate >= 0),
    rate_type          TEXT          NOT NULL CHECK (rate_type IN ('effective_annual', 'nominal_annual', 'monthly')),
    installments       INT           NOT NULL CHECK (installments > 0),
    first_payment_date DATE          NOT NULL,
    notes              TEXT,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_debts_workspace ON debts(workspace_id);

-- ── debt_payments ──────────────────────────────────────────────────────────────
CREATE TABLE debt_payments (
    id         UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    debt_id    UUID          NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
    period     INT           NOT NULL CHECK (period >= 1),
    amount     NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    paid_at    DATE          NOT NULL,
    notes      TEXT,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (debt_id, period)
);

CREATE INDEX idx_debt_payments_debt ON debt_payments(debt_id);

-- ── savings_goals ──────────────────────────────────────────────────────────────
CREATE TABLE savings_goals (
    id            UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id  UUID          NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name          TEXT          NOT NULL,
    target_amount NUMERIC(15,2) NOT NULL CHECK (target_amount > 0),
    deadline      DATE,
    notes         TEXT,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_savings_goals_workspace ON savings_goals(workspace_id);

-- ── savings_contributions ──────────────────────────────────────────────────────
CREATE TABLE savings_contributions (
    id             UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    goal_id        UUID          NOT NULL REFERENCES savings_goals(id) ON DELETE CASCADE,
    amount         NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    contributed_at DATE          NOT NULL,
    notes          TEXT,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_savings_contributions_goal ON savings_contributions(goal_id);
