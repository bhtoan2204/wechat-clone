-- up
BEGIN;

ALTER TABLE relationship_accounts
RENAME TO relationship_account;

ALTER TABLE room_accounts
RENAME TO room_account;

COMMIT;