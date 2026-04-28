package query

import (
	"context"
	"errors"
	"strings"

	"wechat-clone/core/modules/relationship/application/dto/out"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

func mapRelationshipAccountSummaries(
	ctx context.Context,
	accountRepo AccountReadRepository,
	accountIDs []string,
) ([]out.RelationshipAccountSummaryResponse, error) {
	if len(accountIDs) == 0 {
		return []out.RelationshipAccountSummaryResponse{}, nil
	}

	items := make([]out.RelationshipAccountSummaryResponse, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		item, err := loadRelationshipAccountSummary(ctx, accountRepo, accountID)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		items = append(items, item)
	}

	return items, nil
}

func loadRelationshipAccountSummary(
	ctx context.Context,
	accountRepo AccountReadRepository,
	accountID string,
) (out.RelationshipAccountSummaryResponse, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return out.RelationshipAccountSummaryResponse{}, nil
	}
	if accountRepo == nil {
		return out.RelationshipAccountSummaryResponse{AccountID: accountID}, nil
	}

	account, err := accountRepo.GetByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return out.RelationshipAccountSummaryResponse{AccountID: accountID}, nil
		}
		return out.RelationshipAccountSummaryResponse{}, stackErr.Error(err)
	}
	if account == nil {
		return out.RelationshipAccountSummaryResponse{AccountID: accountID}, nil
	}

	return out.RelationshipAccountSummaryResponse{
		AccountID:       accountID,
		DisplayName:     strings.TrimSpace(account.DisplayName),
		Username:        strings.TrimSpace(account.Username),
		AvatarObjectKey: strings.TrimSpace(account.AvatarObjectKey),
	}, nil
}
