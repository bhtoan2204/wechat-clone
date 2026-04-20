UPDATE messages
SET message = COALESCE(file_name, object_key, '[' || message_type || ']')
WHERE message IS NULL;

ALTER TABLE messages
ALTER COLUMN message SET NOT NULL;
