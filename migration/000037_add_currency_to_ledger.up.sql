ALTER TABLE ledger_transactions ADD (
    currency VARCHAR2(32)
);

ALTER TABLE ledger_entries ADD (
    currency VARCHAR2(32)
);

ALTER TRIGGER trg_ledger_transactions_append_only DISABLE;
ALTER TRIGGER trg_ledger_entries_append_only DISABLE;

UPDATE ledger_transactions lt
SET currency = (
    SELECT UPPER(TRIM(pi.currency))
    FROM payment_intents pi
    WHERE pi.transaction_id = REGEXP_REPLACE(lt.transaction_id, '^payment:(.*):succeeded$', '\1')
)
WHERE currency IS NULL
  AND REGEXP_LIKE(lt.transaction_id, '^payment:.*:succeeded$')
  AND EXISTS (
      SELECT 1
      FROM payment_intents pi
      WHERE pi.transaction_id = REGEXP_REPLACE(lt.transaction_id, '^payment:(.*):succeeded$', '\1')
  );

UPDATE ledger_transactions
SET currency = 'UNKNOWN'
WHERE currency IS NULL;

UPDATE ledger_entries le
SET currency = (
    SELECT lt.currency
    FROM ledger_transactions lt
    WHERE lt.transaction_id = le.transaction_id
)
WHERE currency IS NULL;

UPDATE ledger_entries
SET currency = 'UNKNOWN'
WHERE currency IS NULL;

ALTER TABLE ledger_transactions MODIFY (
    currency VARCHAR2(32) NOT NULL
);

ALTER TABLE ledger_entries MODIFY (
    currency VARCHAR2(32) NOT NULL
);

CREATE INDEX idx_ledger_entries_account_currency_created
    ON ledger_entries(account_id, currency, created_at);

ALTER TRIGGER trg_ledger_transactions_append_only ENABLE;
ALTER TRIGGER trg_ledger_entries_append_only ENABLE;
