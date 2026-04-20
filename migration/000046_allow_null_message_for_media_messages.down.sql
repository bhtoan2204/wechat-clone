UPDATE messages
SET message = COALESCE(file_name, object_key, '[' || message_type || ']')
WHERE message IS NULL;

ALTER TABLE messages MODIFY message VARCHAR2(4000) NOT NULL;
