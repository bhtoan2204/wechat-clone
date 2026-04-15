UPDATE payment_intents
SET clearing_account_key = 'provider:' || LOWER(TRIM(provider))
WHERE clearing_account_key IS NULL;
