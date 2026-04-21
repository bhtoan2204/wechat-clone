DROP INDEX IF EXISTS idx_relationship_outbox_aggregate;
DROP INDEX IF EXISTS idx_relationship_outbox_created_at;

ALTER TABLE relationship_outbox_events
    ADD COLUMN event_type VARCHAR(100),
    ADD COLUMN payload JSONB,
    ADD COLUMN occurred_at TIMESTAMPTZ,
    ADD COLUMN published_at TIMESTAMPTZ;

UPDATE relationship_outbox_events
SET
    event_type = event_name,
    payload = event_data::jsonb,
    occurred_at = created_at;

ALTER TABLE relationship_outbox_events
    ALTER COLUMN event_type SET NOT NULL,
    ALTER COLUMN payload SET NOT NULL,
    ALTER COLUMN occurred_at SET NOT NULL,
    ALTER COLUMN aggregate_id TYPE VARCHAR(36),
    ALTER COLUMN aggregate_type TYPE VARCHAR(50);

ALTER TABLE relationship_outbox_events
    DROP COLUMN IF EXISTS version,
    DROP COLUMN IF EXISTS event_name,
    DROP COLUMN IF EXISTS event_data;

CREATE INDEX idx_outbox_unpublished
ON relationship_outbox_events (created_at)
WHERE published_at IS NULL;

ALTER TABLE room_members
    ADD CONSTRAINT fk_room_members_account
    FOREIGN KEY (account_id)
    REFERENCES accounts(id)
    ON DELETE CASCADE;

ALTER TABLE messages
    ADD CONSTRAINT fk_messages_sender
    FOREIGN KEY (sender_id)
    REFERENCES accounts(id)
    ON DELETE CASCADE;

ALTER TABLE message_receipts
    ADD CONSTRAINT fk_message_receipts_account
    FOREIGN KEY (account_id)
    REFERENCES accounts(id)
    ON DELETE CASCADE;

ALTER TABLE message_deletions
    ADD CONSTRAINT fk_message_deletions_account
    FOREIGN KEY (account_id)
    REFERENCES accounts(id)
    ON DELETE CASCADE;
