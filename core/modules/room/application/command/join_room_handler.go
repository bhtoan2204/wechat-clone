package command

import (
	"context"
	"errors"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomsupport "go-socket/core/modules/room/application/support"
	roomrepos "go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
)

type joinRoomHandler struct {
	baseRepo roomrepos.Repos
}

func NewJoinRoomHandler(baseRepo roomrepos.Repos) cqrs.Handler[*in.JoinRoomRequest, *out.JoinRoomResponse] {
	return &joinRoomHandler{
		baseRepo: baseRepo,
	}
}

func (h *joinRoomHandler) Handle(ctx context.Context, req *in.JoinRoomRequest) (*out.JoinRoomResponse, error) {
	_, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return nil, stackErr.Error(errors.New("not implemented"))
}
