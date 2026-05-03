package repository

import (
	"errors"
	shareddb "wechat-clone/core/shared/infra/db"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

var (
	ErrPaymentIntentAggregateNotFound           = errors.New("payment intent aggregate not found")
	ErrPaymentIntentAggregateDuplicateIntent    = errors.New("payment intent aggregate duplicate intent")
	ErrPaymentIntentAggregateDuplicateProcessed = errors.New("payment intent aggregate duplicate processed event")
)

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return stackErr.Error(ErrPaymentIntentAggregateNotFound)
	}
	if shareddb.IsUniqueConstraintError(err) {
		return stackErr.Error(ErrPaymentIntentAggregateDuplicateIntent)
	}
	return stackErr.Error(err)
}
