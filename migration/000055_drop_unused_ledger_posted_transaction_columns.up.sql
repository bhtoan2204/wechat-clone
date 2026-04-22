ALTER TABLE ledger_posted_transactions
    DROP COLUMN IF EXISTS reference_type,
    DROP COLUMN IF EXISTS reference_id,
    DROP COLUMN IF EXISTS counterparty_account_id,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS amount_delta,
    DROP COLUMN IF EXISTS booked_at;
