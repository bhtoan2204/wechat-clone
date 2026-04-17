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

type getChatPresenceHandler struct {
	presence roomservice.PresenceQueryService
}

func NewGetChatPresenceHandler(presence roomservice.PresenceQueryService) cqrs.Handler[*in.GetChatPresenceRequest, *out.ChatPresenceResponse] {
	return &getChatPresenceHandler{presence: presence}
}

func (h *getChatPresenceHandler) Handle(ctx context.Context, req *in.GetChatPresenceRequest) (*out.ChatPresenceResponse, error) {
	res, err := h.presence.GetPresence(ctx, apptypes.GetPresenceQuery{AccountID: req.AccountID})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return roomsupport.ToPresenceResponse(res), nil
}
