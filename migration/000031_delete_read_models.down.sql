CREATE TABLE room_read_models (
    id                    VARCHAR2(1024) PRIMARY KEY,
    name                  VARCHAR2(1024) NOT NULL,
    description           VARCHAR2(4000) DEFAULT '',
    room_type             VARCHAR2(32) NOT NULL,
    owner_id              VARCHAR2(1024) NOT NULL,
    direct_key            VARCHAR2(2048),
    pinned_message_id     VARCHAR2(1024),
    member_count          NUMBER(10) DEFAULT 0 NOT NULL,
    last_message_id       VARCHAR2(1024),
    last_message_at       TIMESTAMP WITH TIME ZONE,
    last_message_content  VARCHAR2(4000),
    last_message_sender_id VARCHAR2(1024),
    created_at            TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at            TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE UNIQUE INDEX uq_room_read_models_direct_key ON room_read_models(direct_key);
CREATE INDEX idx_room_read_models_owner_id ON room_read_models(owner_id);
CREATE INDEX idx_room_read_models_updated_at ON room_read_models(updated_at);
CREATE INDEX idx_room_read_models_last_message_id ON room_read_models(last_message_id);

CREATE TABLE room_member_read_models (
    id               VARCHAR2(1024) PRIMARY KEY,
    room_id          VARCHAR2(1024) NOT NULL,
    account_id       VARCHAR2(1024) NOT NULL,
    role             VARCHAR2(32) DEFAULT 'member',
    last_delivered_at TIMESTAMP WITH TIME ZONE,
    last_read_at     TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT uq_room_member_read_models_room_account UNIQUE (room_id, account_id)
);

CREATE INDEX idx_room_member_read_models_room_id ON room_member_read_models(room_id);
CREATE INDEX idx_room_member_read_models_account_id ON room_member_read_models(account_id);

CREATE TABLE message_read_models (
    id                     VARCHAR2(1024) PRIMARY KEY,
    room_id                VARCHAR2(1024) NOT NULL,
    sender_id              VARCHAR2(1024) NOT NULL,
    message                VARCHAR2(4000) NOT NULL,
    message_type           VARCHAR2(50) DEFAULT 'text' NOT NULL,
    reply_to_message_id    VARCHAR2(1024),
    forwarded_from_message_id VARCHAR2(1024),
    file_name              VARCHAR2(1024),
    file_size              NUMBER(19),
    mime_type              VARCHAR2(255),
    object_key             VARCHAR2(2048),
    edited_at              TIMESTAMP WITH TIME ZONE,
    deleted_for_everyone_at TIMESTAMP WITH TIME ZONE,
    created_at             TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL
);

CREATE INDEX idx_message_read_models_room_id ON message_read_models(room_id);
CREATE INDEX idx_message_read_models_sender_id ON message_read_models(sender_id);
CREATE INDEX idx_message_read_models_created_at ON message_read_models(created_at);
CREATE INDEX idx_message_read_models_reply_to_message_id ON message_read_models(reply_to_message_id);
CREATE INDEX idx_message_read_models_forwarded_from_message_id ON message_read_models(forwarded_from_message_id);

CREATE TABLE message_receipt_read_models (
    id          VARCHAR2(1024) PRIMARY KEY,
    message_id  VARCHAR2(1024) NOT NULL,
    account_id   VARCHAR2(1024) NOT NULL,
    status      VARCHAR2(32) NOT NULL,
    delivered_at TIMESTAMP WITH TIME ZONE,
    seen_at      TIMESTAMP WITH TIME ZONE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT uq_message_receipt_read_models_message_account UNIQUE (message_id, account_id)
);

CREATE INDEX idx_message_receipt_read_models_account_id ON message_receipt_read_models(account_id);
CREATE INDEX idx_message_receipt_read_models_status ON message_receipt_read_models(status);

CREATE TABLE message_deletion_read_models (
    id         VARCHAR2(1024) PRIMARY KEY,
    message_id VARCHAR2(1024) NOT NULL,
    account_id VARCHAR2(1024) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT uq_message_deletion_read_models_message_account UNIQUE (message_id, account_id)
);

CREATE INDEX idx_message_deletion_read_models_account_id ON message_deletion_read_models(account_id);
