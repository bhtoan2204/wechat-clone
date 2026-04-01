package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-socket/core/modules/notification/application/dto/in"
	"go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/modules/notification/domain/entity"
	repos "go-socket/core/modules/notification/domain/repos"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

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

	account := ctx.Value("account")
	if account == nil {
		log.Errorw("account not found in context")
		return nil, stackerr.Error(ErrAccountNotFound)
	}

	payload, ok := account.(*xpaseto.PasetoPayload)
	if !ok {
		log.Errorw("invalid account payload")
		return nil, stackerr.Error(errors.New("invalid account payload"))
	}

	keysBytes, err := json.Marshal(req.Keys)
	if err != nil {
		log.Errorw("marshal keys failed", zap.Error(err))
		return nil, stackerr.Error(fmt.Errorf("marshal subscription keys failed: %w", err))
	}

	subscription := &entity.PushSubscription{
		ID:        uuid.New().String(),
		AccountID: payload.AccountID,
		Endpoint:  req.Endpoint,
		Keys:      string(keysBytes),
	}

	if err := h.pushSubscriptionRepo.UpsertPushSubscription(ctx, subscription); err != nil {
		log.Errorw("save push subscription failed", zap.Error(err))
		return nil, stackerr.Error(ErrSavePushSubscriptionFailed)
	}

	return &out.SavePushSubscriptionResponse{Message: "Push subscription saved"}, nil
}
