package query

import (
	"context"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/cqrs"
)

type listChatConversationsHandler struct {
	chatService *roomservice.ChatQueryService
}

func NewListChatConversationsHandler(chatService *roomservice.ChatQueryService) cqrs.Handler[*in.ListChatConversationsRequest, []*out.ChatConversationResponse] {
	return &listChatConversationsHandler{chatService: chatService}
}

func (h *listChatConversationsHandler) Handle(ctx context.Context, req *in.ListChatConversationsRequest) ([]*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	res, err := h.chatService.ListConversations(ctx, accountID, apptypes.ListConversationsQuery{
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		return nil, err
	}

	outItems := make([]*out.ChatConversationResponse, 0, len(res))
	for _, item := range res {
		copyItem := item
		outItems = append(outItems, roomsupport.ToConversationResponse(&copyItem))
	}

	return outItems, nil
}
