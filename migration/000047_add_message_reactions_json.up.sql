ALTER TABLE messages
ADD COLUMN reactions_json TEXT NOT NULL DEFAULT '[]';