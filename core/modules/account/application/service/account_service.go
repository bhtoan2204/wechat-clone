package service

import (
	"context"
	"errors"
	"go-socket/core/modules/account/domain/aggregate"
	"go-socket/core/modules/account/domain/repos"
	"go-socket/core/modules/account/domain/rules"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

//go:generate mockgen -package=service -destination=account_service_mock.go -source=account_service.go
type AccountService interface {
	LoadAccountAggregate(ctx context.Context, accountID string) (*aggregate.AccountAggregate, error)
}

type accountService struct {
	baseRepo repos.Repos
}

func NewAccountService(repos repos.Repos) AccountService {
	return &accountService{
		baseRepo: repos,
	}
}

func (s *accountService) LoadAccountAggregate(ctx context.Context, accountID string) (*aggregate.AccountAggregate, error) {
	accountAggregate, err := s.baseRepo.AccountAggregateRepository().Load(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, stackErr.Error(rules.ErrAccountNotFound)
		}
		return nil, stackErr.Error(err)
	}
	return accountAggregate, nil
}
