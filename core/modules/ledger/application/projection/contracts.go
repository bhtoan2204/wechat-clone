package projection

import (
	"context"
)

type LedgerProjection interface {
	ProjectTransaction(ctx context.Context, transaction *LedgerTransactionProjected) error
}
