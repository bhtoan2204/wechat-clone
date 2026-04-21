-- up
BEGIN;

ALTER TABLE relationship_account
RENAME TO relationship_accounts;

ALTER TABLE room_account
RENAME TO room_accounts;

COMMIT;