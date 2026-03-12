-- =========================
-- TABLE: payment_account_projections
-- =========================
CREATE TABLE payment_account_projections (
    id          VARCHAR2(1024) PRIMARY KEY,
    account_id  VARCHAR2(1024) NOT NULL,
    email       VARCHAR2(1024) NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE INDEX idx_payment_account_projections_account_id ON payment_account_projections(account_id);