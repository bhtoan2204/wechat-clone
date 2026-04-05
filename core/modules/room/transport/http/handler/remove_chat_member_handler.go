// CODE_GENERATOR: handler
package handler

import (
	"errors"
	roomin "go-socket/core/modules/room/application/dto/in"
	roomout "go-socket/core/modules/room/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type removeChatMemberHandler struct {
	removeChatMember cqrs.Dispatcher[*roomin.RemoveChatMemberRequest, *roomout.ChatConversationResponse]
}

func NewRemoveChatMemberHandler(removeChatMember cqrs.Dispatcher[*roomin.RemoveChatMemberRequest, *roomout.ChatConversationResponse]) *removeChatMemberHandler {
	return &removeChatMemberHandler{
		removeChatMember: removeChatMember,
	}
}

func (h *removeChatMemberHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	request := roomin.RemoveChatMemberRequest{
		RoomID:    c.Param("room_id"),
		AccountID: c.Param("account_id"),
	}
	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("validate request failed"))
	}
	result, err := h.removeChatMember.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("RemoveChatMember failed", zap.Error(err))
		return nil, stackerr.Error(errors.New("RemoveChatMember failed"))
	}
	return result, nil
}
