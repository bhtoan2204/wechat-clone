ALTER TABLE payment_intents
    ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 0 NOT NULL;

UPDATE payment_intents pi
SET version = COALESCE((
    SELECT MAX(poe.version)
    FROM payment_outbox_events poe
    WHERE poe.aggregate_id = pi.transaction_id
      AND poe.aggregate_type = 'PaymentIntentAggregate'
), 0);
