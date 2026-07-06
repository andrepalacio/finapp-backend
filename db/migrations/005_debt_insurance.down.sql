ALTER TABLE debts
    DROP COLUMN IF EXISTS insurance_rate,
    DROP COLUMN IF EXISTS insurance_type;
