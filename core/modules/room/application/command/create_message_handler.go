package command

import (
	"context"
	"errors"
	"fmt"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/domain/aggregate"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/cqrs"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"reflect"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type createMessageHandler struct {
	baseRepo repos.Repos
}

func NewCreateMessageHandler(baseRepo repos.Repos) cqrs.Handler[*in.CreateMessageRequest, *out.CreateMessageResponse] {
	return &createMessageHandler{
		baseRepo: baseRepo,
	}
}

func (h *createMessageHandler) Handle(ctx context.Context, req *in.CreateMessageRequest) (*out.CreateMessageResponse, error) {
	log := logging.FromContext(ctx).Named("CreateMessage")
	account := ctx.Value("account")
	if account == nil {
		log.Errorw("Account not found", zap.Error(errors.New("account not found")))
		return nil, stackerr.Error(errors.New("account not found"))
	}
	payload, ok := account.(*xpaseto.PasetoPayload)
	if !ok {
		return nil, stackerr.Error(errors.New("invalid account payload"))
	}

	message := &entity.MessageEntity{
		ID:        uuid.NewString(),
		RoomID:    req.RoomID,
		SenderID:  payload.AccountID,
		Message:   req.Message,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.MessageRepository().CreateMessage(ctx, message); err != nil {
			return fmt.Errorf("create message failed: %w", err)
		}

		roomAggregate := &aggregate.RoomAggregate{}
		roomAggregateType := reflect.TypeOf(roomAggregate).Elem().Name()
		roomAggregate.SetAggregateType(roomAggregateType)
		if err := roomAggregate.SetID(message.RoomID); err != nil {
			return fmt.Errorf("set room aggregate id failed: %w", err)
		}

		if err := roomAggregate.ApplyChange(roomAggregate, &aggregate.EventRoomMessageCreated{
			RoomID:             message.RoomID,
			MessageID:          message.ID,
			MessageContent:     message.Message,
			MessageSenderID:    message.SenderID,
			MessageSenderName:  payload.Email,
			MessageSenderEmail: payload.Email,
			MessageSentAt:      message.CreatedAt,
		}); err != nil {
			return fmt.Errorf("apply room message created event failed: %w", err)
		}

		publisher := eventpkg.NewPublisher(txRepos.RoomOutboxEventsRepository())
		if err := publisher.PublishAggregate(ctx, roomAggregate); err != nil {
			return fmt.Errorf("publish room message created event failed: %w", err)
		}

		return nil
	}); err != nil {
		log.Errorw("Failed to create message", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	return &out.CreateMessageResponse{
		Id:        message.ID,
		RoomId:    message.RoomID,
		SenderId:  message.SenderID,
		Message:   message.Message,
		CreatedAt: message.CreatedAt.Format(time.RFC3339),
	}, nil
}
