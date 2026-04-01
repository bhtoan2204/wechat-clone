package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate value")
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
