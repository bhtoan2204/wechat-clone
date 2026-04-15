DROP INDEX idx_ledger_entries_account_currency_created;

ALTER TABLE ledger_entries DROP COLUMN currency;
ALTER TABLE ledger_transactions DROP COLUMN currency;
