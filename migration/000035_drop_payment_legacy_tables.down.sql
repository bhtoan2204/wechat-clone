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

-- =========================
-- TABLE: payment_aggregates
-- =========================
CREATE TABLE payment_aggregates (
    id             VARCHAR2(1024) PRIMARY KEY,
    aggregate_id   VARCHAR2(1024) NOT NULL,
    aggregate_type VARCHAR2(255)  NOT NULL,
    version        NUMBER(10)     NOT NULL,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE UNIQUE INDEX idx_payment_aggregates_aggregate_id ON payment_aggregates(aggregate_id);
CREATE INDEX idx_payment_aggregates_aggregate_type ON payment_aggregates(aggregate_type);
CREATE INDEX idx_payment_aggregates_version ON payment_aggregates(version);

-- =========================
-- TABLE: payment_balances
-- =========================
CREATE TABLE payment_balances (
    id          VARCHAR2(1024) PRIMARY KEY,
    account_id  VARCHAR2(1024) NOT NULL,
    amount      NUMBER(19)     NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE INDEX idx_payment_balances_account_id ON payment_balances(account_id);

-- =========================
-- TABLE: payment_balance_snapshots
-- =========================
CREATE TABLE payment_balance_snapshots (
    id            VARCHAR2(1024) PRIMARY KEY,
    aggregate_id  VARCHAR2(1024) NOT NULL,
    version       NUMBER(10)     NOT NULL,
    state         CLOB           NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE UNIQUE INDEX idx_snap_ver ON payment_balance_snapshots(aggregate_id, version);

-- =========================
-- TABLE: payment_events
-- =========================
CREATE TABLE payment_events (
    id             VARCHAR2(1024) PRIMARY KEY,
    aggregate_id   VARCHAR2(1024) NOT NULL,
    aggregate_type VARCHAR2(255)  NOT NULL,
    version        NUMBER(10)     NOT NULL,
    event_name     VARCHAR2(255)  NOT NULL,
    event_data     CLOB           NOT NULL,
    metadata       CLOB           NOT NULL,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

GRANT SELECT ON payment_events TO C##DBZUSER;
ALTER TABLE payment_events ADD SUPPLEMENTAL LOG DATA (ALL) COLUMNS;

CREATE UNIQUE INDEX idx_agg_ver ON payment_events(aggregate_id, version);
CREATE INDEX idx_payment_events_event_name ON payment_events(event_name);

-- =========================
-- TABLE: payment_event_offsets
-- =========================
CREATE TABLE payment_event_offsets (
    consumer_name VARCHAR2(1024) PRIMARY KEY,
    last_event_id NUMBER(19)     NOT NULL,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

-- =========================
-- TABLE: payment_transactions
-- =========================
CREATE TABLE payment_transactions (
    id          VARCHAR2(1024) PRIMARY KEY,
    account_id  VARCHAR2(1024) NOT NULL,
    event_id    VARCHAR2(1024) NOT NULL,
    amount      NUMBER(19)     NOT NULL,
    type        VARCHAR2(255)  NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE INDEX idx_payment_transactions_account_id ON payment_transactions(account_id);
CREATE INDEX idx_payment_transactions_event_id ON payment_transactions(event_id);

-- =========================
-- TABLE: payment_histories
-- =========================
CREATE TABLE payment_histories (
    id            VARCHAR2(36) PRIMARY KEY,
    type          VARCHAR2(50)  NOT NULL,
    amount        NUMBER(19)    NOT NULL,
    balance       NUMBER(19)    NOT NULL,
    sender_id     VARCHAR2(36),
    receiver_id   VARCHAR2(36),
    sender_name   VARCHAR2(255),
    receiver_name VARCHAR2(255),
    properties    CLOB          NOT NULL,
    created_at    TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE INDEX idx_payment_histories_type ON payment_histories(type);
CREATE INDEX idx_payment_histories_sender_id ON payment_histories(sender_id);
CREATE INDEX idx_payment_histories_receiver_id ON payment_histories(receiver_id);
CREATE INDEX idx_payment_histories_sender_created ON payment_histories(sender_id, created_at);
CREATE INDEX idx_payment_histories_receiver_created ON payment_histories(receiver_id, created_at);
