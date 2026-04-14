-- devices
CREATE TABLE devices (
    id               VARCHAR2(36 CHAR)                        NOT NULL,
    account_id       VARCHAR2(1024 CHAR)                      NOT NULL,
    device_uid       VARCHAR2(128 CHAR)                       NOT NULL,
    device_name      VARCHAR2(200 CHAR),
    device_type      VARCHAR2(30 CHAR)    DEFAULT 'web'       NOT NULL,
    os_name          VARCHAR2(50 CHAR),
    os_version       VARCHAR2(50 CHAR),
    app_version      VARCHAR2(50 CHAR),
    user_agent       VARCHAR2(1000 CHAR),
    last_ip_address  VARCHAR2(45 CHAR),
    last_seen_at     TIMESTAMP(6) WITH TIME ZONE,
    is_trusted       NUMBER(1,0)          DEFAULT 0           NOT NULL,
    created_at       TIMESTAMP(6) WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at       TIMESTAMP(6) WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,

    CONSTRAINT pk_devices      PRIMARY KEY (id),
    CONSTRAINT fk_dev_acc      FOREIGN KEY (account_id)
                               REFERENCES accounts(id)
                               ON DELETE CASCADE,
    CONSTRAINT uk_dev_acc_uid  UNIQUE (account_id, device_uid),
    -- Required so sessions can enforce account-safe device ownership.
    CONSTRAINT uk_dev_acc_id   UNIQUE (account_id, id),
    CONSTRAINT ck_dev_type     CHECK (device_type IN ('web', 'ios', 'android', 'desktop', 'other')),
    CONSTRAINT ck_dev_trusted  CHECK (is_trusted IN (0, 1))
);

CREATE INDEX ix_dev_seen ON devices (last_seen_at);

CREATE OR REPLACE TRIGGER trg_devices_bu
BEFORE UPDATE ON devices
FOR EACH ROW
BEGIN
    :NEW.updated_at := SYSTIMESTAMP;
END;
/
--------------------------------------------------------------------------------

-- sessions
CREATE TABLE sessions (
    id                 VARCHAR2(36 CHAR)                        NOT NULL,
    account_id         VARCHAR2(1024 CHAR)                      NOT NULL,
    device_id          VARCHAR2(36 CHAR)                        NOT NULL,
    refresh_token_hash VARCHAR2(255 CHAR)                       NOT NULL,
    status             VARCHAR2(20 CHAR)    DEFAULT 'active'    NOT NULL,
    ip_address         VARCHAR2(45 CHAR),
    user_agent         VARCHAR2(1000 CHAR),
    last_activity_at   TIMESTAMP(6) WITH TIME ZONE,
    expires_at         TIMESTAMP(6) WITH TIME ZONE              NOT NULL,
    revoked_at         TIMESTAMP(6) WITH TIME ZONE,
    revoked_reason     VARCHAR2(255 CHAR),
    created_at         TIMESTAMP(6) WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at         TIMESTAMP(6) WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,

    CONSTRAINT pk_sessions   PRIMARY KEY (id),
    CONSTRAINT fk_ses_dev    FOREIGN KEY (account_id, device_id)
                             REFERENCES devices(account_id, id)
                             ON DELETE CASCADE,
    CONSTRAINT uk_ses_rth    UNIQUE (refresh_token_hash),
    CONSTRAINT ck_ses_st     CHECK (status IN ('active', 'revoked', 'expired'))
);

CREATE INDEX ix_ses_acc_st   ON sessions (account_id, status);
CREATE INDEX ix_ses_acc_dev  ON sessions (account_id, device_id);
CREATE INDEX ix_ses_exp      ON sessions (expires_at);
CREATE INDEX ix_ses_last_act ON sessions (last_activity_at);

CREATE OR REPLACE TRIGGER trg_sessions_bu
BEFORE UPDATE ON sessions
FOR EACH ROW
BEGIN
    :NEW.updated_at := SYSTIMESTAMP;
END;
/
