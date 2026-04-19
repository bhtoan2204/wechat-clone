package socket

import (
	"context"
	"time"

	"wechat-clone/core/shared/pkg/logging"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type client struct {
	id       string
	accountID string
	conn     *websocket.Conn
	sendCh   chan []byte
}

func newClient(conn *websocket.Conn, clientID, accountID string) *client {
	if clientID == "" {
		clientID = accountID + ":" + time.Now().UTC().Format(time.RFC3339Nano)
	}
	return &client{
		id:        clientID,
		accountID: accountID,
		conn:      conn,
		sendCh:    make(chan []byte, 256),
	}
}

func (c *client) readPump(ctx context.Context, unregister func(context.Context, *client)) {
	log := logging.FromContext(ctx)
	defer unregister(ctx, c)
	defer c.close(ctx)

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Warnw("unexpected notification websocket close", "client_id", c.id, zap.Error(err))
			}
			return
		}
	}
}

func (c *client) writePump(ctx context.Context) {
	log := logging.FromContext(ctx)
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer c.close(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-c.sendCh:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				log.Warnw("write notification websocket message failed", "client_id", c.id, zap.Error(err))
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warnw("write notification websocket ping failed", "client_id", c.id, zap.Error(err))
				return
			}
		}
	}
}

func (c *client) send(_ context.Context, payload []byte) {
	select {
	case c.sendCh <- payload:
	default:
	}
}

func (c *client) close(ctx context.Context) {
	select {
	case <-ctx.Done():
	default:
	}
	_ = c.conn.Close()
}
