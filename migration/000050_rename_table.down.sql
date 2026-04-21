-- down
BEGIN;

ALTER TABLE relationship_account
RENAME TO relationship_account_projections;

ALTER TABLE room_account
RENAME TO room_account_projections;

COMMIT;