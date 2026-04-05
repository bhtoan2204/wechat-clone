package command

import (
	"context"
	"errors"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type createMessageHandler struct {
	messageService *roomservice.MessageCommandService
}

func NewCreateMessageHandler(messageService *roomservice.MessageCommandService) cqrs.Handler[*in.CreateMessageRequest, *out.CreateMessageResponse] {
	return &createMessageHandler{messageService: messageService}
}

func (h *createMessageHandler) Handle(ctx context.Context, req *in.CreateMessageRequest) (*out.CreateMessageResponse, error) {
	log := logging.FromContext(ctx).Named("CreateMessage")
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		log.Errorw("Account not found", zap.Error(err))
		return nil, stackerr.Error(errors.New("account not found"))
	}

	res, err := h.messageService.CreateMessage(ctx, accountID, apptypes.SendMessageCommand{
		RoomID:  req.RoomID,
		Message: req.Message,
	})
	if err != nil {
		log.Errorw("Failed to create message", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	return &out.CreateMessageResponse{
		Id:        res.ID,
		RoomId:    res.RoomID,
		SenderId:  res.SenderID,
		Message:   res.Message,
		CreatedAt: res.CreatedAt,
	}, nil
}
