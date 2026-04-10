package socket

import (
	"context"
	"net/http"

	"go-socket/core/shared/pkg/actorctx"
	"go-socket/core/shared/pkg/logging"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type wsHandler struct {
	hub      IHub
	upgrader websocket.Upgrader
}

func NewWSHandler(hub IHub) *wsHandler {
	return &wsHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *wsHandler) Handle(c *gin.Context) {
	ctx := c.Request.Context()
	log := logging.FromContext(ctx)

	if h.hub == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "websocket hub is not initialized"})
		return
	}

	accountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorw("failed to upgrade websocket connection", zap.Error(err))
		return
	}

	client := NewClient(ctx, conn, c.Query("client_id"), accountID)
	h.hub.Register(ctx, client)

	clientCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go client.WritePump(clientCtx)
	client.ReadPump(clientCtx, h.hub)
}
