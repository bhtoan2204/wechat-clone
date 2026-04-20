ALTER TABLE room_members DROP CONSTRAINT fk_room_members_account;
ALTER TABLE messages DROP CONSTRAINT fk_messages_sender;
ALTER TABLE message_receipts DROP CONSTRAINT fk_message_receipts_account;
ALTER TABLE message_deletions DROP CONSTRAINT fk_message_deletions_account;
