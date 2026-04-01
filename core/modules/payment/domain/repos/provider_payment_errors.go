package repos

import "errors"

var (
	ErrProviderPaymentNotFound           = errors.New("provider payment not found")
	ErrProviderPaymentDuplicateIntent    = errors.New("provider payment duplicate intent")
	ErrProviderPaymentDuplicateProcessed = errors.New("provider payment duplicate processed event")
)
