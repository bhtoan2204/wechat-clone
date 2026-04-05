ALTER TABLE rooms ADD direct_key VARCHAR2(2048);
ALTER TABLE rooms ADD pinned_message_id VARCHAR2(1024);
CREATE UNIQUE INDEX uq_rooms_direct_key ON rooms(direct_key);

ALTER TABLE room_members ADD last_delivered_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE room_members ADD last_read_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE messages ADD message_type VARCHAR2(50) DEFAULT 'text' NOT NULL;
ALTER TABLE messages ADD reply_to_message_id VARCHAR2(1024);
ALTER TABLE messages ADD forwarded_from_message_id VARCHAR2(1024);
ALTER TABLE messages ADD file_name VARCHAR2(1024);
ALTER TABLE messages ADD file_size NUMBER(19);
ALTER TABLE messages ADD mime_type VARCHAR2(255);
ALTER TABLE messages ADD object_key VARCHAR2(2048);
ALTER TABLE messages ADD edited_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE messages ADD deleted_for_everyone_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX idx_messages_reply_to_message_id ON messages(reply_to_message_id);
CREATE INDEX idx_messages_forwarded_from_message_id ON messages(forwarded_from_message_id);

CREATE TABLE message_receipts (
    id            VARCHAR2(1024) PRIMARY KEY,
    message_id    VARCHAR2(1024) NOT NULL,
    account_id    VARCHAR2(1024) NOT NULL,
    status        VARCHAR2(32) NOT NULL,
    delivered_at  TIMESTAMP WITH TIME ZONE,
    seen_at       TIMESTAMP WITH TIME ZONE,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT fk_message_receipts_message
        FOREIGN KEY (message_id)
        REFERENCES messages(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_message_receipts_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id)
        ON DELETE CASCADE,
    CONSTRAINT uq_message_receipts_message_account UNIQUE (message_id, account_id)
);

CREATE INDEX idx_message_receipts_account_id ON message_receipts(account_id);
CREATE INDEX idx_message_receipts_status ON message_receipts(status);

CREATE TABLE message_deletions (
    id          VARCHAR2(1024) PRIMARY KEY,
    message_id  VARCHAR2(1024) NOT NULL,
    account_id  VARCHAR2(1024) NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT fk_message_deletions_message
        FOREIGN KEY (message_id)
        REFERENCES messages(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_message_deletions_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id)
        ON DELETE CASCADE,
    CONSTRAINT uq_message_deletions_message_account UNIQUE (message_id, account_id)
);

CREATE INDEX idx_message_deletions_account_id ON message_deletions(account_id);
