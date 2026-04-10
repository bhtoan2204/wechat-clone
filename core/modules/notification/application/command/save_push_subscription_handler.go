package command

import (
	"context"
	"encoding/json"
	"fmt"
	"go-socket/core/modules/notification/application/dto/in"
	"go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/modules/notification/domain/entity"
	repos "go-socket/core/modules/notification/domain/repos"
	"go-socket/core/shared/pkg/actorctx"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type savePushSubscriptionHandler struct {
	pushSubscriptionRepo repos.PushSubscriptionRepository
}

func NewSavePushSubscriptionHandler(baseRepo repos.Repos) cqrs.Handler[*in.SavePushSubscriptionRequest, *out.SavePushSubscriptionResponse] {
	return &savePushSubscriptionHandler{pushSubscriptionRepo: baseRepo.PushSubscriptionRepository()}
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

	subscription := &entity.PushSubscription{
		ID:        uuid.New().String(),
		AccountID: accountID,
		Endpoint:  req.Endpoint,
		Keys:      string(keysBytes),
	}

	if err := h.pushSubscriptionRepo.UpsertPushSubscription(ctx, subscription); err != nil {
		log.Errorw("save push subscription failed", zap.Error(err))
		return nil, stackErr.Error(ErrSavePushSubscriptionFailed)
	}

	return &out.SavePushSubscriptionResponse{Message: "Push subscription saved"}, nil
}
