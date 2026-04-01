package assembly

import "errors"

var (
	ErrMissingLedgerGateway = errors.New("payment ledger gateway is not registered")
	ErrInvalidLedgerGateway = errors.New("payment ledger gateway has invalid type")
)
