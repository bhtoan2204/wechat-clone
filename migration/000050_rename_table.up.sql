-- up
BEGIN;

ALTER TABLE relationship_account_projections
RENAME TO relationship_account;

ALTER TABLE room_account_projections
RENAME TO room_account;

COMMIT;