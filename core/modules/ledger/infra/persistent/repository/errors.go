package repository

import (
	"errors"
	"strings"

	ledgerrepos "go-socket/core/modules/ledger/domain/repos"

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
		return ErrNotFound
	}

	if isOracleUniqueConstraintError(err) {
		return ErrDuplicate
	}
	return err
}

func isOracleUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToUpper(err.Error()), "ORA-00001")
}
