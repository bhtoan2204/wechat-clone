package projection

import (
	appprojection "wechat-clone/core/modules/ledger/application/projection"
	"wechat-clone/core/modules/ledger/infra/projection/repository"

	"gorm.io/gorm"
)

func NewLedgerProjector(session *gorm.DB) (appprojection.LedgerProjection, error) {
	store := repository.NewLedgerRepoImpl(session)
	return store, nil
}

func NewLedgerReadRepository(session *gorm.DB) (appprojection.ReadRepository, error) {
	store := repository.NewLedgerRepoImpl(session)
	return store, nil
}
