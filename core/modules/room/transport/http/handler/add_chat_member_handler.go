// CODE_GENERATOR: handler
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

type addChatMemberHandler struct {
	addChatMember cqrs.Dispatcher[*in.AddChatMemberRequest, *out.ChatConversationResponse]
}

func NewAddChatMemberHandler(
	addChatMember cqrs.Dispatcher[*in.AddChatMemberRequest, *out.ChatConversationResponse],
) *addChatMemberHandler {
	return &addChatMemberHandler{
		addChatMember: addChatMember,
	}
}

func (h *addChatMemberHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.AddChatMemberRequest
	request.RoomID = c.Param("room_id")
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Errorw("Unmarshal request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil
	}

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, nil
	}

	result, err := h.addChatMember.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("AddChatMember failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
