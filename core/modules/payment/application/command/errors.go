package command

import "errors"

var (
	ErrPaymentAccountNotFound  = errors.New("account not found")
	ErrInvalidPaymentAmount    = errors.New("invalid payment amount")
	ErrInsufficientBalance     = errors.New("insufficient balance")
	ErrPaymentVersionConflict  = errors.New("payment aggregate version conflict")
	ErrPaymentReceiverNotFound = errors.New("receiver account not found")
)
