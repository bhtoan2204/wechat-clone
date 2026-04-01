package repository

import (
	"errors"
	"strings"

	paymentrepos "go-socket/core/modules/payment/domain/repos"

	"gorm.io/gorm"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return paymentrepos.ErrProviderPaymentNotFound
	}
	if isOracleUniqueConstraintError(err) {
		return paymentrepos.ErrProviderPaymentDuplicateIntent
	}
	return err
}

func isOracleUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToUpper(err.Error()), "ORA-00001")
}
