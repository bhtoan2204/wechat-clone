// CODE_GENERATOR - do not edit: handler
package handler

import (
	"net/http"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type removeChatMemberHandler struct {
	removeChatMember cqrs.Dispatcher[*in.RemoveChatMemberRequest, *out.ChatConversationResponse]
}

func NewRemoveChatMemberHandler(
	removeChatMember cqrs.Dispatcher[*in.RemoveChatMemberRequest, *out.ChatConversationResponse],
) *removeChatMemberHandler {
	return &removeChatMemberHandler{
		removeChatMember: removeChatMember,
	}
}

func (h *removeChatMemberHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.RemoveChatMemberRequest
	request.RoomID = c.Param("room_id")
	request.AccountID = c.Param("account_id")

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	result, err := h.removeChatMember.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("RemoveChatMember failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
