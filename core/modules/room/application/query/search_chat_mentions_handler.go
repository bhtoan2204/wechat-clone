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

type searchChatMentionsHandler struct {
	chatService *roomservice.ChatQueryService
}

func NewSearchChatMentionsHandler(chatService *roomservice.ChatQueryService) cqrs.Handler[*in.SearchChatMentionsRequest, []*out.ChatMentionCandidateResponse] {
	return &searchChatMentionsHandler{chatService: chatService}
}

func (h *searchChatMentionsHandler) Handle(ctx context.Context, req *in.SearchChatMentionsRequest) ([]*out.ChatMentionCandidateResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	results, err := h.chatService.SearchMentionCandidates(ctx, accountID, apptypes.SearchMentionCandidatesQuery{
		RoomID: req.RoomID,
		Query:  req.Query,
		Limit:  req.Limit,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	outItems := make([]*out.ChatMentionCandidateResponse, 0, len(results))
	for _, item := range results {
		copyItem := item
		outItems = append(outItems, &out.ChatMentionCandidateResponse{
			AccountID:       copyItem.AccountID,
			DisplayName:     copyItem.DisplayName,
			Username:        copyItem.Username,
			AvatarObjectKey: copyItem.AvatarObjectKey,
		})
	}
	return outItems, nil
}
