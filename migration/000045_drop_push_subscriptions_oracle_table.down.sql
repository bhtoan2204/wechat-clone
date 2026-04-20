CREATE TABLE push_subscriptions (
    id          VARCHAR2(1024) PRIMARY KEY,
    account_id  VARCHAR2(1024) NOT NULL,
    endpoint    VARCHAR2(2048) NOT NULL,
    keys        CLOB NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE INDEX idx_push_subscriptions_account_id ON push_subscriptions(account_id);
CREATE UNIQUE INDEX uq_push_subscriptions_account_endpoint
ON push_subscriptions(account_id, endpoint);
