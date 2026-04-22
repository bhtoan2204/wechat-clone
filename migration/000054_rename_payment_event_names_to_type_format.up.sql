UPDATE payment_outbox_events
SET event_name = CASE event_name
    WHEN 'payment.created' THEN 'EventPaymentCreated'
    WHEN 'payment.checkout_session_created' THEN 'EventPaymentCheckoutSessionCreated'
    WHEN 'payment.succeeded' THEN 'EventPaymentSucceeded'
    WHEN 'payment.failed' THEN 'EventPaymentFailed'
    WHEN 'payment.refunded' THEN 'EventPaymentRefunded'
    WHEN 'payment.chargeback' THEN 'EventPaymentChargeback'
    ELSE event_name
END
WHERE event_name IN (
    'payment.created',
    'payment.checkout_session_created',
    'payment.succeeded',
    'payment.failed',
    'payment.refunded',
    'payment.chargeback'
);

UPDATE processed_payment_events
SET idempotency_key = CASE
    WHEN idempotency_key LIKE 'payment.succeeded:%' THEN REPLACE(idempotency_key, 'payment.succeeded:', 'EventPaymentSucceeded:')
    WHEN idempotency_key LIKE 'payment.refunded:%' THEN REPLACE(idempotency_key, 'payment.refunded:', 'EventPaymentRefunded:')
    WHEN idempotency_key LIKE 'payment.chargeback:%' THEN REPLACE(idempotency_key, 'payment.chargeback:', 'EventPaymentChargeback:')
    WHEN idempotency_key LIKE 'payment.failed:%' THEN REPLACE(idempotency_key, 'payment.failed:', 'EventPaymentFailed:')
    ELSE idempotency_key
END
WHERE idempotency_key LIKE 'payment.succeeded:%'
   OR idempotency_key LIKE 'payment.refunded:%'
   OR idempotency_key LIKE 'payment.chargeback:%'
   OR idempotency_key LIKE 'payment.failed:%';
