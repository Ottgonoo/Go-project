CREATE TABLE entries (
    id BIGSERIAL PRIMARY KEY,

    transaction_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,

    direction VARCHAR(10) NOT NULL,

    amount BIGINT NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_entry_transaction
        FOREIGN KEY (transaction_id)
        REFERENCES transactions(id)
        ON DELETE RESTRICT,

    CONSTRAINT fk_entry_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id)
        ON DELETE RESTRICT,

    CONSTRAINT valid_entry_direction
        CHECK (direction IN ('DEBIT', 'CREDIT')),

    CONSTRAINT positive_entry_amount
        CHECK (amount > 0)
);