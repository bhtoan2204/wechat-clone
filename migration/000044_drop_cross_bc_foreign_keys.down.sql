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
