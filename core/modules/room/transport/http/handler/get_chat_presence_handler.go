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

type getChatPresenceHandler struct {
	getChatPresence cqrs.Dispatcher[*in.GetChatPresenceRequest, *out.ChatPresenceResponse]
}

func NewGetChatPresenceHandler(
	getChatPresence cqrs.Dispatcher[*in.GetChatPresenceRequest, *out.ChatPresenceResponse],
) *getChatPresenceHandler {
	return &getChatPresenceHandler{
		getChatPresence: getChatPresence,
	}
}

func (h *getChatPresenceHandler) Handle(c *gin.Context) (interface{}, error) {
	ctx := c.Request.Context()
	logger := logging.FromContext(ctx)
	var request in.GetChatPresenceRequest
	request.AccountID = c.Param("account_id")

	if err := request.Validate(); err != nil {
		logger.Errorw("Validate request failed", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, stackErr.Error(err)
	}

	result, err := h.getChatPresence.Dispatch(ctx, &request)
	if err != nil {
		logger.Errorw("GetChatPresence failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return result, nil
}
