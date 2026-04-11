package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-socket/core/modules/notification/application/dto/in"
	"go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/modules/notification/domain/aggregate"
	repos "go-socket/core/modules/notification/domain/repos"
	"go-socket/core/shared/pkg/actorctx"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type savePushSubscriptionHandler struct {
	baseRepo repos.Repos
}

func NewSavePushSubscriptionHandler(baseRepo repos.Repos) cqrs.Handler[*in.SavePushSubscriptionRequest, *out.SavePushSubscriptionResponse] {
	return &savePushSubscriptionHandler{baseRepo: baseRepo}
}

func (h *savePushSubscriptionHandler) Handle(ctx context.Context, req *in.SavePushSubscriptionRequest) (*out.SavePushSubscriptionResponse, error) {
	log := logging.FromContext(ctx).Named("SavePushSubscription")

	accountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		log.Errorw("account not found in context")
		return nil, stackErr.Error(ErrAccountNotFound)
	}

	keysBytes, err := json.Marshal(req.Keys)
	if err != nil {
		log.Errorw("marshal keys failed", zap.Error(err))
		return nil, stackErr.Error(fmt.Errorf("marshal subscription keys failed: %v", err))
	}

	if err := h.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		now := time.Now().UTC()
		pushSubscriptionRepo := txRepos.PushSubscriptionRepository()
		subscriptionAgg, err := pushSubscriptionRepo.LoadByAccountAndEndpoint(ctx, accountID, req.Endpoint)
		if err != nil {
			if !errors.Is(err, repos.ErrPushSubscriptionNotFound) {
				return stackErr.Error(err)
			}

			subscriptionAgg, err = aggregate.NewPushSubscriptionAggregate(uuid.New().String())
			if err != nil {
				return stackErr.Error(err)
			}
			if err := subscriptionAgg.Create(accountID, req.Endpoint, string(keysBytes), now); err != nil {
				return stackErr.Error(err)
			}
		} else {
			changed, err := subscriptionAgg.UpdateKeys(string(keysBytes), now)
			if err != nil {
				return stackErr.Error(err)
			}
			if !changed {
				return nil
			}
		}

		return pushSubscriptionRepo.Save(ctx, subscriptionAgg)
	}); err != nil {
		log.Errorw("save push subscription failed", zap.Error(err))
		return nil, stackErr.Error(ErrSavePushSubscriptionFailed)
	}

	return &out.SavePushSubscriptionResponse{Message: "Push subscription saved"}, nil
}
