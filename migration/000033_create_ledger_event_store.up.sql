CREATE TABLE ledger_aggregates (
    id             VARCHAR2(1024) PRIMARY KEY,
    aggregate_id   VARCHAR2(1024) NOT NULL,
    aggregate_type VARCHAR2(255)  NOT NULL,
    version        NUMBER(10)     NOT NULL,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE UNIQUE INDEX idx_ledger_aggregates_aggregate_id
    ON ledger_aggregates(aggregate_id);
CREATE INDEX idx_ledger_aggregates_aggregate_type
    ON ledger_aggregates(aggregate_type);
CREATE INDEX idx_ledger_aggregates_version
    ON ledger_aggregates(version);

CREATE TABLE ledger_events (
    id             VARCHAR2(1024) PRIMARY KEY,
    aggregate_id   VARCHAR2(1024) NOT NULL,
    aggregate_type VARCHAR2(255)  NOT NULL,
    version        NUMBER(10)     NOT NULL,
    event_name     VARCHAR2(255)  NOT NULL,
    event_data     CLOB           NOT NULL,
    metadata       CLOB           NOT NULL,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

GRANT SELECT ON ledger_events TO C##DBZUSER;
ALTER TABLE ledger_events ADD SUPPLEMENTAL LOG DATA (ALL) COLUMNS;

CREATE UNIQUE INDEX idx_ledger_events_agg_ver
    ON ledger_events(aggregate_id, version);
CREATE INDEX idx_ledger_events_event_name
    ON ledger_events(event_name);
