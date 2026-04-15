package projection

import "context"

//go:generate mockgen -package=projection -destination=projector_mock.go -source=projector.go
type Projector interface {
	ProjectTransaction(ctx context.Context, transaction *LedgerTransactionProjected) error
}
