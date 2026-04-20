CREATE TABLE relationship_account_projections (
    account_id           VARCHAR(36) PRIMARY KEY,
    display_name         VARCHAR(255) NOT NULL DEFAULT '',
    username             VARCHAR(255) NOT NULL DEFAULT '',
    avatar_object_key    VARCHAR(2048) NOT NULL DEFAULT '',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_relationship_account_projections_updated_at
ON relationship_account_projections (updated_at DESC);
