package query

import (
	"context"

	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	roomservice "wechat-clone/core/modules/room/application/service"
	roomsupport "wechat-clone/core/modules/room/application/support"
	apptypes "wechat-clone/core/modules/room/application/types"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type getChatConversationMetadataHandler struct {
	conversations roomservice.ConversationQueryService
}

func NewGetChatConversationMetadataHandler(conversations roomservice.ConversationQueryService) cqrs.Handler[*in.GetChatConversationRequest, *out.ChatConversationMetadataResponse] {
	return &getChatConversationMetadataHandler{conversations: conversations}
}

func (h *getChatConversationMetadataHandler) Handle(ctx context.Context, req *in.GetChatConversationRequest) (*out.ChatConversationMetadataResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	res, err := h.conversations.GetConversationMetadata(ctx, accountID, apptypes.GetConversationQuery{
		RoomID: req.RoomID,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return roomsupport.ToConversationMetadataResponse(res), nil
}
