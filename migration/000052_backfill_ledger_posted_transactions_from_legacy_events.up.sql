ALTER TABLE ledger_posted_transactions
    ADD COLUMN IF NOT EXISTS event_name VARCHAR(255);

ALTER TABLE ledger_posted_transactions
    ADD COLUMN IF NOT EXISTS event_data TEXT;

ALTER TABLE ledger_posted_transactions
    ALTER COLUMN reference_type DROP NOT NULL,
    ALTER COLUMN reference_id DROP NOT NULL,
    ALTER COLUMN counterparty_account_id DROP NOT NULL,
    ALTER COLUMN currency DROP NOT NULL,
    ALTER COLUMN amount_delta DROP NOT NULL,
    ALTER COLUMN booked_at DROP NOT NULL;

INSERT INTO ledger_posted_transactions (
    id,
    aggregate_id,
    aggregate_type,
    transaction_id,
    event_name,
    event_data,
    created_at
)
SELECT
    LOWER(md5(random()::text || clock_timestamp()::text)) AS id,
    aggregate_id,
    aggregate_type,
    (event_data::jsonb ->> 'transaction_id') AS transaction_id,
    event_name,
    event_data,
    created_at
FROM ledger_events
WHERE aggregate_type = 'LedgerAccountAggregate'
  AND event_name IN (
      'EventLedgerAccountPaymentBooked',
      'EventLedgerAccountDepositFromIntent',
      'EventLedgerAccountWithdrawFromIntent',
      'EventLedgerAccountDepositFromRefund',
      'EventLedgerAccountWithdrawFromRefund',
      'EventLedgerAccountDepositFromChargeback',
      'EventLedgerAccountWithdrawFromChargeback',
      'EventLedgerAccountTransferredToAccount',
      'EventLedgerAccountReceivedTransfer'
  )
ON CONFLICT (aggregate_id, aggregate_type, transaction_id) DO UPDATE
SET
    event_name = EXCLUDED.event_name,
    event_data = EXCLUDED.event_data;
