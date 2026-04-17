package query

import (
	"context"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
)

type listChatConversationsHandler struct {
	conversations roomservice.ConversationQueryService
}

func NewListChatConversationsHandler(conversations roomservice.ConversationQueryService) cqrs.Handler[*in.ListChatConversationsRequest, []*out.ChatConversationResponse] {
	return &listChatConversationsHandler{conversations: conversations}
}

func (h *listChatConversationsHandler) Handle(ctx context.Context, req *in.ListChatConversationsRequest) ([]*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	res, err := h.conversations.ListConversations(ctx, accountID, apptypes.ListConversationsQuery{
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	outItems := make([]*out.ChatConversationResponse, 0, len(res))
	for _, item := range res {
		copyItem := item
		outItems = append(outItems, roomsupport.ToConversationResponse(&copyItem))
	}

	return outItems, nil
}
