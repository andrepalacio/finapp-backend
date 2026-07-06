ALTER TABLE debts
    ADD COLUMN insurance_rate NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN insurance_type TEXT          NOT NULL DEFAULT '';
