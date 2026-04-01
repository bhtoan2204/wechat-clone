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

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type createRoomHandler struct {
	baseRepo repos.Repos
}

func NewCreateRoomHandler(baseRepo repos.Repos) cqrs.Handler[*in.CreateRoomRequest, *out.CreateRoomResponse] {
	return &createRoomHandler{
		baseRepo: baseRepo,
	}
}

func (h *createRoomHandler) Handle(ctx context.Context, req *in.CreateRoomRequest) (*out.CreateRoomResponse, error) {
	log := logging.FromContext(ctx).Named("CreateRoom")
	account := ctx.Value("account").(*xpaseto.PasetoPayload)
	if account == nil {
		log.Errorw("Account not found", zap.Error(errors.New("account not found")))
		return nil, stackerr.Error(errors.New("account not found"))
	}
	room := &entity.Room{
		ID:          uuid.NewString(),
		Name:        req.Name,
		Description: req.Description,
		RoomType:    req.RoomType,
		OwnerID:     account.AccountID,
	}

	if err := h.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.RoomRepository().CreateRoom(ctx, room); err != nil {
			return fmt.Errorf("create room failed: %w", err)
		}

		roomAggregate := &aggregate.RoomAggregate{}
		roomAggregateType := reflect.TypeOf(roomAggregate).Elem().Name()
		roomAggregate.SetAggregateType(roomAggregateType)
		if err := roomAggregate.SetID(room.ID); err != nil {
			return fmt.Errorf("set room aggregate id failed: %w", err)
		}

		if err := roomAggregate.ApplyChange(roomAggregate, &aggregate.EventRoomCreated{
			RoomID:      room.ID,
			RoomType:    room.RoomType,
			MemberCount: 1,
		}); err != nil {
			return fmt.Errorf("apply room created event failed: %w", err)
		}

		publisher := eventpkg.NewPublisher(txRepos.RoomOutboxEventsRepository())
		if err := publisher.PublishAggregate(ctx, roomAggregate); err != nil {
			return fmt.Errorf("publish room created event failed: %w", err)
		}

		return nil
	}); err != nil {
		log.Errorw("Failed to create room", zap.Error(err), zap.Any("room", room))
		return nil, stackerr.Error(err)
	}

	return &out.CreateRoomResponse{
		Id:   room.ID,
		Name: room.Name,
	}, nil
}
