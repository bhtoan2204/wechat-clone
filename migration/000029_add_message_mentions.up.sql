ALTER TABLE messages ADD mentions_json CLOB DEFAULT '[]' NOT NULL;
ALTER TABLE messages ADD mention_all NUMBER(1) DEFAULT 0 NOT NULL;
ALTER TABLE messages ADD CONSTRAINT chk_messages_mention_all CHECK (mention_all IN (0, 1));

ALTER TABLE message_read_models ADD mentions_json CLOB DEFAULT '[]' NOT NULL;
ALTER TABLE message_read_models ADD mention_all NUMBER(1) DEFAULT 0 NOT NULL;
ALTER TABLE message_read_models ADD CONSTRAINT chk_msg_read_models_mention_all CHECK (mention_all IN (0, 1));
