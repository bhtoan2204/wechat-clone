ALTER TABLE message_read_models DROP CONSTRAINT chk_msg_read_models_mention_all;
ALTER TABLE message_read_models DROP COLUMN mention_all;
ALTER TABLE message_read_models DROP COLUMN mentions_json;

ALTER TABLE messages DROP CONSTRAINT chk_messages_mention_all;
ALTER TABLE messages DROP COLUMN mention_all;
ALTER TABLE messages DROP COLUMN mentions_json;
