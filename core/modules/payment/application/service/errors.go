package service

import "errors"

var (
	ErrValidation            = errors.New("validation failed")
	ErrDuplicateTransaction  = errors.New("transaction already exists")
	ErrTransactionNotFound   = errors.New("transaction not found")
	ErrDuplicatePayment      = errors.New("payment already exists")
	ErrPaymentIntentNotFound = errors.New("payment intent not found")
)
