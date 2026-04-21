CREATE TABLE ledger_posted_transactions (
    id                      VARCHAR(1024) PRIMARY KEY,
    aggregate_id            VARCHAR(1024) NOT NULL,
    aggregate_type          VARCHAR(255)  NOT NULL,
    transaction_id          VARCHAR(1024) NOT NULL,
    reference_type          VARCHAR(255)  NOT NULL,
    reference_id            VARCHAR(1024) NOT NULL,
    counterparty_account_id VARCHAR(1024) NOT NULL,
    currency                VARCHAR(16)   NOT NULL,
    amount_delta            BIGINT     NOT NULL,
    booked_at               TIMESTAMPTZ NOT NULL,
    created_at              TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO ledger_posted_transactions (
    id,
    aggregate_id,
    aggregate_type,
    transaction_id,
    reference_type,
    reference_id,
    counterparty_account_id,
    currency,
    amount_delta,
    booked_at,
    created_at
)
SELECT
    LOWER(md5(random()::text || clock_timestamp()::text)) AS id,
    aggregate_id,
    aggregate_type,
    (event_data::jsonb ->> 'transaction_id') AS transaction_id,
    CASE event_name
        WHEN 'EventLedgerAccountPaymentBooked' THEN COALESCE((event_data::jsonb ->> 'reference_type'), 'payment.succeeded')
        WHEN 'EventLedgerAccountDepositFromIntent' THEN 'payment.succeeded'
        WHEN 'EventLedgerAccountWithdrawFromIntent' THEN 'payment.succeeded'
        WHEN 'EventLedgerAccountDepositFromRefund' THEN 'payment.refunded'
        WHEN 'EventLedgerAccountWithdrawFromRefund' THEN 'payment.refunded'
        WHEN 'EventLedgerAccountDepositFromChargeback' THEN 'payment.chargeback'
        WHEN 'EventLedgerAccountWithdrawFromChargeback' THEN 'payment.chargeback'
        ELSE 'ledger.transfer_to_account'
    END AS reference_type,
    CASE event_name
        WHEN 'EventLedgerAccountPaymentBooked' THEN (event_data::jsonb ->> 'payment_id')
        WHEN 'EventLedgerAccountDepositFromIntent' THEN (event_data::jsonb ->> 'payment_id')
        WHEN 'EventLedgerAccountWithdrawFromIntent' THEN (event_data::jsonb ->> 'payment_id')
        WHEN 'EventLedgerAccountDepositFromRefund' THEN (event_data::jsonb ->> 'payment_id')
        WHEN 'EventLedgerAccountWithdrawFromRefund' THEN (event_data::jsonb ->> 'payment_id')
        WHEN 'EventLedgerAccountDepositFromChargeback' THEN (event_data::jsonb ->> 'payment_id')
        WHEN 'EventLedgerAccountWithdrawFromChargeback' THEN (event_data::jsonb ->> 'payment_id')
        ELSE (event_data::jsonb ->> 'transaction_id')
    END AS reference_id,
    CASE event_name
        WHEN 'EventLedgerAccountPaymentBooked' THEN (event_data::jsonb ->> 'counterparty_account_id')
        WHEN 'EventLedgerAccountDepositFromIntent' THEN (event_data::jsonb ->> 'counterparty_account_id')
        WHEN 'EventLedgerAccountWithdrawFromIntent' THEN (event_data::jsonb ->> 'counterparty_account_id')
        WHEN 'EventLedgerAccountDepositFromRefund' THEN (event_data::jsonb ->> 'counterparty_account_id')
        WHEN 'EventLedgerAccountWithdrawFromRefund' THEN (event_data::jsonb ->> 'counterparty_account_id')
        WHEN 'EventLedgerAccountDepositFromChargeback' THEN (event_data::jsonb ->> 'counterparty_account_id')
        WHEN 'EventLedgerAccountWithdrawFromChargeback' THEN (event_data::jsonb ->> 'counterparty_account_id')
        WHEN 'EventLedgerAccountTransferredToAccount' THEN (event_data::jsonb ->> 'to_account_id')
        WHEN 'EventLedgerAccountReceivedTransfer' THEN (event_data::jsonb ->> 'from_account_id')
    END AS counterparty_account_id,
    UPPER((event_data::jsonb ->> 'currency')) AS currency,
    CASE event_name
        WHEN 'EventLedgerAccountPaymentBooked' THEN NULLIF((event_data::jsonb ->> 'amount_delta'), '')::BIGINT
        WHEN 'EventLedgerAccountDepositFromIntent' THEN NULLIF((event_data::jsonb ->> 'amount'), '')::BIGINT
        WHEN 'EventLedgerAccountWithdrawFromIntent' THEN -NULLIF((event_data::jsonb ->> 'amount'), '')::BIGINT
        WHEN 'EventLedgerAccountDepositFromRefund' THEN NULLIF((event_data::jsonb ->> 'amount'), '')::BIGINT
        WHEN 'EventLedgerAccountWithdrawFromRefund' THEN -NULLIF((event_data::jsonb ->> 'amount'), '')::BIGINT
        WHEN 'EventLedgerAccountDepositFromChargeback' THEN NULLIF((event_data::jsonb ->> 'amount'), '')::BIGINT
        WHEN 'EventLedgerAccountWithdrawFromChargeback' THEN -NULLIF((event_data::jsonb ->> 'amount'), '')::BIGINT
        WHEN 'EventLedgerAccountTransferredToAccount' THEN -NULLIF((event_data::jsonb ->> 'amount'), '')::BIGINT
        WHEN 'EventLedgerAccountReceivedTransfer' THEN NULLIF((event_data::jsonb ->> 'amount'), '')::BIGINT
    END AS amount_delta,
    created_at AS booked_at,
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
  );

CREATE UNIQUE INDEX idx_ledger_posted_tx_agg_type_tx
    ON ledger_posted_transactions(aggregate_id, aggregate_type, transaction_id);
