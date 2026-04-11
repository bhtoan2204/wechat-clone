// CODE_GENERATOR: application-handler
package query

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/application/service"
	repos "go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
)

type listRoomsHandler struct {
}

func NewListRooms(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
	roomQueryService *service.RoomQueryService,
) cqrs.Handler[*in.ListRoomsRequest, *out.ListRoomsResponse] {
	return &listRoomsHandler{}
}

func (u *listRoomsHandler) Handle(ctx context.Context, req *in.ListRoomsRequest) (*out.ListRoomsResponse, error) {
	return nil, stackErr.Error(fmt.Errorf("not implemented yet"))
}
