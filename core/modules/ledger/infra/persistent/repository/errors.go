package repository

import (
	"errors"
	"strings"

	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

var (
	ErrNotFound  = ledgerrepos.ErrNotFound
	ErrDuplicate = ledgerrepos.ErrDuplicate
)

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return stackErr.Error(ErrNotFound)
	}

	if isOracleUniqueConstraintError(err) {
		return stackErr.Error(ErrDuplicate)
	}
	return stackErr.Error(err)
}

func isOracleUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToUpper(err.Error()), "ORA-00001")
}
