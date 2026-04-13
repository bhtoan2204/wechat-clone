// CODE_GENERATOR: application-handler
package query

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/application/projection"
	"go-socket/core/modules/room/application/service"
	"go-socket/core/shared/pkg/cqrs"
)

type listRoomsHandler struct {
}

func NewListRooms(
	appCtx *appCtx.AppContext,
	readRepo projection.QueryRepos,
	roomQueryService *service.RoomQueryService,
) cqrs.Handler[*in.ListRoomsRequest, *out.ListRoomsResponse] {
	return &listRoomsHandler{}
}

func (u *listRoomsHandler) Handle(ctx context.Context, req *in.ListRoomsRequest) (*out.ListRoomsResponse, error) {
	return nil, fmt.Errorf("not implemented yet")
}
