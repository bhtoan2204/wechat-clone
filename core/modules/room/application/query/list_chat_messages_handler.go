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

type listChatMessagesHandler struct {
	chatService *roomservice.ChatQueryService
}

func NewListChatMessagesHandler(chatService *roomservice.ChatQueryService) cqrs.Handler[*in.ListChatMessagesRequest, []*out.ChatMessageResponse] {
	return &listChatMessagesHandler{chatService: chatService}
}

func (h *listChatMessagesHandler) Handle(ctx context.Context, req *in.ListChatMessagesRequest) ([]*out.ChatMessageResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	res, err := h.chatService.ListMessages(ctx, accountID, apptypes.ListMessagesQuery{
		RoomID:    req.RoomID,
		Limit:     req.Limit,
		BeforeID:  req.BeforeID,
		BeforeAt:  req.BeforeAt,
		Ascending: req.Ascending,
	})
	if err != nil {
		return nil, err
	}

	outItems := make([]*out.ChatMessageResponse, 0, len(res))
	for _, item := range res {
		copyItem := item
		outItems = append(outItems, roomsupport.ToMessageResponse(&copyItem))
	}

	return outItems, nil
}
