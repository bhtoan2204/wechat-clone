DROP INDEX idx_message_deletions_account_id;
DROP TABLE message_deletions CASCADE CONSTRAINTS;

DROP INDEX idx_message_receipts_status;
DROP INDEX idx_message_receipts_account_id;
DROP TABLE message_receipts CASCADE CONSTRAINTS;

DROP INDEX idx_messages_forwarded_from_message_id;
DROP INDEX idx_messages_reply_to_message_id;

ALTER TABLE messages DROP COLUMN deleted_for_everyone_at;
ALTER TABLE messages DROP COLUMN edited_at;
ALTER TABLE messages DROP COLUMN object_key;
ALTER TABLE messages DROP COLUMN mime_type;
ALTER TABLE messages DROP COLUMN file_size;
ALTER TABLE messages DROP COLUMN file_name;
ALTER TABLE messages DROP COLUMN forwarded_from_message_id;
ALTER TABLE messages DROP COLUMN reply_to_message_id;
ALTER TABLE messages DROP COLUMN message_type;

ALTER TABLE room_members DROP COLUMN last_read_at;
ALTER TABLE room_members DROP COLUMN last_delivered_at;

DROP INDEX uq_rooms_direct_key;
ALTER TABLE rooms DROP COLUMN pinned_message_id;
ALTER TABLE rooms DROP COLUMN direct_key;
