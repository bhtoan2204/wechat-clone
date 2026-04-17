package command

import (
	"context"
	"reflect"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	roomrepos "go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
)

type forwardChatMessageHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewForwardChatMessageHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.ForwardChatMessageRequest, *out.ChatMessageResponse] {
	return &forwardChatMessageHandler{baseRepo: baseRepo, realtime: realtime}
}

func (h *forwardChatMessageHandler) Handle(ctx context.Context, req *in.ForwardChatMessageRequest) (*out.ChatMessageResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	sourceMessage, err := h.baseRepo.MessageAggregateRepository().Load(ctx, req.MessageID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	res, err := executeSendMessage(ctx, h.baseRepo, accountID, apptypes.SendMessageCommand{
		RoomID:                 req.TargetRoomID,
		Message:                sourceMessage.Message().Message,
		MessageType:            sourceMessage.Message().MessageType,
		ForwardedFromMessageID: sourceMessage.Message().ID,
		FileName:               sourceMessage.Message().FileName,
		FileSize:               sourceMessage.Message().FileSize,
		MimeType:               sourceMessage.Message().MimeType,
		ObjectKey:              sourceMessage.Message().ObjectKey,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	out := roomsupport.ToMessageResponse(res)
	h.realtime.EmitMessage(ctx, types.MessagePayload{
		RoomId:  out.RoomID,
		Type:    reflect.TypeOf(out).Elem().Name(),
		Payload: out,
	})
	return out, nil
}
