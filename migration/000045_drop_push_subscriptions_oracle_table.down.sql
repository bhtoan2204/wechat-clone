CREATE TABLE push_subscriptions (
    id          VARCHAR(1024) PRIMARY KEY,
    account_id  VARCHAR(1024) NOT NULL,
    endpoint    VARCHAR(2048) NOT NULL,
    keys        TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_push_subscriptions_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_push_subscriptions_account_id ON push_subscriptions(account_id);
CREATE UNIQUE INDEX uq_push_subscriptions_account_endpoint
ON push_subscriptions(account_id, endpoint);
