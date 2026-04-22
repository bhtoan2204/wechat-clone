ALTER TABLE ledger_posted_transactions
    DROP COLUMN IF EXISTS event_name;

ALTER TABLE ledger_posted_transactions
    DROP COLUMN IF EXISTS event_data;
