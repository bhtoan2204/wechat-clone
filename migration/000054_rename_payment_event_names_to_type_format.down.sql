UPDATE payment_outbox_events
SET event_name = CASE event_name
    WHEN 'EventPaymentCreated' THEN 'payment.created'
    WHEN 'EventPaymentCheckoutSessionCreated' THEN 'payment.checkout_session_created'
    WHEN 'EventPaymentSucceeded' THEN 'payment.succeeded'
    WHEN 'EventPaymentFailed' THEN 'payment.failed'
    WHEN 'EventPaymentRefunded' THEN 'payment.refunded'
    WHEN 'EventPaymentChargeback' THEN 'payment.chargeback'
    ELSE event_name
END
WHERE event_name IN (
    'EventPaymentCreated',
    'EventPaymentCheckoutSessionCreated',
    'EventPaymentSucceeded',
    'EventPaymentFailed',
    'EventPaymentRefunded',
    'EventPaymentChargeback'
);

UPDATE processed_payment_events
SET idempotency_key = CASE
    WHEN idempotency_key LIKE 'EventPaymentSucceeded:%' THEN REPLACE(idempotency_key, 'EventPaymentSucceeded:', 'payment.succeeded:')
    WHEN idempotency_key LIKE 'EventPaymentRefunded:%' THEN REPLACE(idempotency_key, 'EventPaymentRefunded:', 'payment.refunded:')
    WHEN idempotency_key LIKE 'EventPaymentChargeback:%' THEN REPLACE(idempotency_key, 'EventPaymentChargeback:', 'payment.chargeback:')
    WHEN idempotency_key LIKE 'EventPaymentFailed:%' THEN REPLACE(idempotency_key, 'EventPaymentFailed:', 'payment.failed:')
    ELSE idempotency_key
END
WHERE idempotency_key LIKE 'EventPaymentSucceeded:%'
   OR idempotency_key LIKE 'EventPaymentRefunded:%'
   OR idempotency_key LIKE 'EventPaymentChargeback:%'
   OR idempotency_key LIKE 'EventPaymentFailed:%';
