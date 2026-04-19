package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const roomChannelPrefix = "room:"

type roomSubscription struct {
	pubsub *redis.PubSub
	cancel context.CancelFunc
	once   sync.Once
}

func (s *roomSubscription) Close() error {
	var closeErr error
	s.once.Do(func() {
		s.cancel()
		closeErr = s.pubsub.Close()
	})
	return closeErr
}

type Hub struct {
	redisClient *redis.Client

	mu            sync.RWMutex
	clients       map[string]*client
	rooms         map[string]*room
	clientRooms   map[string]map[string]struct{}
	subscriptions map[string]*roomSubscription
	closed        bool
}

func NewHub(appCtx *appCtx.AppContext) *Hub {
	return &Hub{
		redisClient:   appCtx.GetRedisClient(),
		clients:       make(map[string]*client),
		rooms:         make(map[string]*room),
		clientRooms:   make(map[string]map[string]struct{}),
		subscriptions: make(map[string]*roomSubscription),
	}
}

func (h *Hub) Register(c *client) {
	if c == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.id] = c
	if _, ok := h.clientRooms[c.id]; !ok {
		h.clientRooms[c.id] = make(map[string]struct{})
	}
}

func (h *Hub) Unregister(ctx context.Context, c *client) {
	if c == nil {
		return
	}
	h.mu.Lock()
	delete(h.clients, c.id)
	roomIDs := h.clientRooms[c.id]
	delete(h.clientRooms, c.id)
	h.mu.Unlock()

	for roomID := range roomIDs {
		h.leaveRoom(ctx, c, roomID)
	}
}

func (h *Hub) JoinRoom(ctx context.Context, c *client, roomID string) error {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return stackErr.Error(fmt.Errorf("room_id is required"))
	}

	h.mu.Lock()
	r, ok := h.rooms[roomID]
	if !ok {
		r = newRoom(roomID)
		h.rooms[roomID] = r
	}
	r.addClient(c)
	if _, ok := h.clientRooms[c.id]; !ok {
		h.clientRooms[c.id] = make(map[string]struct{})
	}
	h.clientRooms[c.id][roomID] = struct{}{}
	_, subscribed := h.subscriptions[roomID]
	h.mu.Unlock()

	if !subscribed {
		if err := h.subscribeRoom(ctx, roomID); err != nil {
			return stackErr.Error(err)
		}
	}
	return nil
}

func (h *Hub) leaveRoom(ctx context.Context, c *client, roomID string) {
	h.mu.Lock()
	r := h.rooms[roomID]
	if r != nil {
		r.removeClient(c.id)
		if r.isEmpty() {
			delete(h.rooms, roomID)
		}
	}
	if rooms := h.clientRooms[c.id]; rooms != nil {
		delete(rooms, roomID)
	}
	sub := h.subscriptions[roomID]
	shouldClose := r == nil || r.isEmpty()
	if shouldClose {
		delete(h.subscriptions, roomID)
	}
	h.mu.Unlock()

	if shouldClose && sub != nil {
		_ = sub.Close()
	}
}

func (h *Hub) Publish(ctx context.Context, msg Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal notification socket message failed: %w", err))
	}
	if err := h.redisClient.Publish(ctx, roomChannelName(msg.RoomID), payload).Err(); err != nil {
		return stackErr.Error(fmt.Errorf("publish notification room message failed: %w", err))
	}
	return nil
}

func (h *Hub) Close(ctx context.Context) {
	log := logging.FromContext(ctx)
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	clients := make([]*client, 0, len(h.clients))
	for _, c := range h.clients {
		clients = append(clients, c)
	}
	subs := make([]*roomSubscription, 0, len(h.subscriptions))
	for _, sub := range h.subscriptions {
		subs = append(subs, sub)
	}
	h.clients = map[string]*client{}
	h.rooms = map[string]*room{}
	h.clientRooms = map[string]map[string]struct{}{}
	h.subscriptions = map[string]*roomSubscription{}
	h.mu.Unlock()

	for _, sub := range subs {
		if err := sub.Close(); err != nil {
			log.Warnw("close notification room subscription failed", zap.Error(err))
		}
	}
	for _, c := range clients {
		c.close(ctx)
	}
}

func (h *Hub) subscribeRoom(ctx context.Context, roomID string) error {
	subCtx, cancel := context.WithCancel(ctx)
	pubsub := h.redisClient.Subscribe(subCtx, roomChannelName(roomID))
	sub := &roomSubscription{pubsub: pubsub, cancel: cancel}

	if _, err := pubsub.Receive(subCtx); err != nil {
		_ = sub.Close()
		return stackErr.Error(fmt.Errorf("subscribe notification room failed: %w", err))
	}

	h.mu.Lock()
	h.subscriptions[roomID] = sub
	h.mu.Unlock()

	go h.consumeRoomMessages(subCtx, roomID, sub)
	return nil
}

func (h *Hub) consumeRoomMessages(ctx context.Context, roomID string, sub *roomSubscription) {
	channel := sub.pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-channel:
			if !ok {
				return
			}
			h.broadcast(roomID, []byte(item.Payload))
		}
	}
}

func (h *Hub) broadcast(roomID string, payload []byte) {
	h.mu.RLock()
	r := h.rooms[roomID]
	h.mu.RUnlock()
	if r == nil {
		return
	}
	r.broadcast(context.Background(), payload)
}

func roomChannelName(roomID string) string {
	roomID = strings.TrimSpace(roomID)
	roomID = strings.TrimPrefix(roomID, roomChannelPrefix)
	return roomChannelPrefix + roomID
}
