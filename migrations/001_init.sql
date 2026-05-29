-- Accounts table
CREATE TABLE IF NOT EXISTS accounts (
    account_id      BIGSERIAL    PRIMARY KEY,
    document_number VARCHAR(50)  NOT NULL UNIQUE
);

-- Operation types lookup table
CREATE TABLE IF NOT EXISTS operation_types (
    operation_type_id BIGINT       PRIMARY KEY,
    description       VARCHAR(100) NOT NULL
);

-- Seed operation types
INSERT INTO operation_types (operation_type_id, description)
VALUES
    (1, 'Normal Purchase'),
    (2, 'Purchase with installments'),
    (3, 'Withdrawal'),
    (4, 'Credit Voucher')
ON CONFLICT DO NOTHING;

-- Transactions table
-- type (debit/credit) is set by the application layer based on operation_type_id
CREATE TABLE IF NOT EXISTS transactions (
    transaction_id    BIGSERIAL      PRIMARY KEY,
    account_id        BIGINT         NOT NULL REFERENCES accounts(account_id),
    operation_type_id BIGINT         NOT NULL REFERENCES operation_types(operation_type_id),
    amount            NUMERIC(15, 2) NOT NULL CHECK (amount <> 0),
    type              VARCHAR(6)     NOT NULL,
    event_date        TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- FK constraints do NOT create indexes in Postgres automatically.
-- This index makes queries filtering by account_id efficient.
CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions (account_id);
