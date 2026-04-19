package service

import (
	"context"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/notification/constant"
	"wechat-clone/core/modules/notification/types"
	"wechat-clone/core/shared/pkg/pubsub"
	"wechat-clone/core/shared/pkg/stackErr"
)

//go:generate mockgen -package=service -destination=realtime_service_mock.go -source=realtime_service.go
type RealtimeService interface {
	EmitMessage(ctx context.Context, message types.RealtimeMessagePayload) error
}

type realtimeService struct {
	localPublisher *pubsub.Bus
}

func NewRealtimeService(appCtx *appCtx.AppContext) RealtimeService {
	return &realtimeService{
		localPublisher: appCtx.LocalBus(),
	}
}

func (s *realtimeService) EmitMessage(ctx context.Context, message types.RealtimeMessagePayload) error {
	if err := s.localPublisher.Publish(ctx, constant.RealtimeMessageTopic, message); err != nil {
		return stackErr.Error(err)
	}
	return nil
}
