package command

import (
	"context"

	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	"wechat-clone/core/modules/room/application/service"
	roomsupport "wechat-clone/core/modules/room/application/support"
	apptypes "wechat-clone/core/modules/room/application/types"
	roomrepos "wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type forwardChatMessageHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewForwardChatMessageHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.ForwardChatMessageRequest, *out.ChatMessageCommandResponse] {
	return &forwardChatMessageHandler{baseRepo: baseRepo, realtime: realtime}
}

func (h *forwardChatMessageHandler) Handle(ctx context.Context, req *in.ForwardChatMessageRequest) (*out.ChatMessageCommandResponse, error) {
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

	return &out.ChatMessageCommandResponse{MessageID: res.ID, RoomID: res.RoomID, Status: CommandStatusCreated}, nil
}
