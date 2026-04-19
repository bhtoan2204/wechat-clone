package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/notification/constant"
	notificationtypes "wechat-clone/core/modules/notification/types"
	"wechat-clone/core/shared/pkg/actorctx"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/pubsub"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type wsHandler struct {
	hub        *Hub
	upgrader   websocket.Upgrader
	subscriber *pubsub.Subscription
}

func NewWSHandler(appContext *appCtx.AppContext, hub *Hub, upgrader websocket.Upgrader) *wsHandler {
	log := logging.FromContext(context.Background())
	subscriber, err := appContext.LocalBus().Subscribe(constant.RealtimeMessageTopic)
	if err != nil {
		log.Errorw("subscribe notification realtime topic failed", zap.Error(err))
		return nil
	}

	handler := &wsHandler{
		hub:        hub,
		upgrader:   upgrader,
		subscriber: subscriber,
	}

	go handler.consumeRealtimeMessages(context.Background())
	return handler
}

func (h *wsHandler) Handle(c *gin.Context) {
	ctx := c.Request.Context()
	accountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logging.FromContext(ctx).Errorw("upgrade notification websocket failed", zap.Error(err))
		return
	}

	client := newClient(conn, c.Query("client_id"), accountID)
	h.hub.Register(client)
	if err := h.hub.JoinRoom(ctx, client, notificationRoomID(accountID)); err != nil {
		logging.FromContext(ctx).Errorw("join notification room failed", zap.Error(err))
		client.close(ctx)
		return
	}

	clientCtx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		client.readPump(clientCtx, h.hub.Unregister)
	}()
	go func() {
		defer cancel()
		client.writePump(clientCtx)
	}()
}

func (h *wsHandler) Close(ctx context.Context) {
	if h.subscriber != nil {
		h.subscriber.Unsubscribe()
	}
	if h.hub != nil {
		h.hub.Close(ctx)
	}
}

func (h *wsHandler) consumeRealtimeMessages(ctx context.Context) {
	if h == nil || h.subscriber == nil || h.hub == nil {
		return
	}
	log := logging.FromContext(ctx)
	for msg := range h.subscriber.C() {
		if err := handleRealtimeMessage(ctx, h.hub, msg); err != nil {
			log.Warnw("handle notification realtime message failed", zap.Error(err))
		}
	}
}

func handleRealtimeMessage(ctx context.Context, hub *Hub, msg pubsub.Message) error {
	if hub == nil {
		return stackErr.Error(fmt.Errorf("notification hub is nil"))
	}
	if msg.Topic != constant.RealtimeMessageTopic {
		return nil
	}

	payload, ok := msg.Data.(notificationtypes.RealtimeMessagePayload)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid notification realtime payload: %T", msg.Data))
	}

	data, err := json.Marshal(payload.Payload)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal notification realtime payload failed: %w", err))
	}

	return stackErr.Error(hub.Publish(ctx, Message{
		RoomID: payload.RoomID,
		Action: payload.Type,
		Data:   data,
	}))
}

func notificationRoomID(accountID string) string {
	return "notification:" + accountID
}
