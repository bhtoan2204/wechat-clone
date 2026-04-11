package socket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const roomChannelPrefix = "room:"
const presenceTTL = 2 * time.Minute

var _ IHub = (*Hub)(nil)

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
	clients       map[string]IClient
	rooms         map[string]IRoom
	clientRooms   map[string]map[string]struct{}
	subscriptions map[string]*roomSubscription

	closeMu  sync.Once
	isClosed bool
}

func NewHub(ctx context.Context, appCtx *appCtx.AppContext) *Hub {
	return &Hub{
		redisClient:   appCtx.GetRedisClient(),
		clients:       make(map[string]IClient),
		rooms:         make(map[string]IRoom),
		clientRooms:   make(map[string]map[string]struct{}),
		subscriptions: make(map[string]*roomSubscription),
	}
}

func (h *Hub) Register(ctx context.Context, client IClient) {
	log := logging.FromContext(ctx)
	if client == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.isClosed {
		client.Close(ctx)
		return
	}
	h.clients[client.GetID()] = client
	if _, ok := h.clientRooms[client.GetID()]; !ok {
		h.clientRooms[client.GetID()] = make(map[string]struct{})
	}
	h.publishPresence(ctx, client.GetUserID(), "online")

	log.Infow("client registered", "client_id", client.GetID(), "user_id", client.GetUserID(), "clients", len(h.clients))
}

func (h *Hub) Unregister(ctx context.Context, client IClient) {
	log := logging.FromContext(ctx)
	if client == nil {
		return
	}

	clientID := client.GetID()

	h.mu.Lock()
	if _, exists := h.clients[clientID]; !exists {
		h.mu.Unlock()
		client.Close(ctx)
		return
	}
	delete(h.clients, clientID)
	roomIDs := make([]string, 0, len(h.clientRooms[clientID]))
	for roomID := range h.clientRooms[clientID] {
		roomIDs = append(roomIDs, roomID)
	}
	h.mu.Unlock()

	for _, roomID := range roomIDs {
		if err := h.LeaveRoom(ctx, client, roomID); err != nil {
			log.Warnw("failed to leave room while unregistering client", "client_id", clientID, "room_id", roomID, zap.Error(err))
		}
	}

	h.mu.Lock()
	delete(h.clientRooms, clientID)
	remainingClients := len(h.clients)
	h.mu.Unlock()

	h.publishPresence(ctx, client.GetUserID(), "offline")
	client.Close(ctx)
	log.Infow("client unregistered", "client_id", clientID, "clients", remainingClients)
}

func (h *Hub) JoinRoom(ctx context.Context, client IClient, roomID string) error {
	log := logging.FromContext(ctx)
	if client == nil {
		return stackErr.Error(errors.New("client is nil"))
	}
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}

	h.mu.Lock()
	if h.isClosed {
		h.mu.Unlock()
		return stackErr.Error(errors.New("hub is closed"))
	}
	if _, ok := h.clients[client.GetID()]; !ok {
		h.clients[client.GetID()] = client
	}
	room, ok := h.rooms[roomID]
	if !ok {
		room = NewRoom(ctx, roomID)
		h.rooms[roomID] = room
	}
	room.AddClient(ctx, client)

	if _, ok := h.clientRooms[client.GetID()]; !ok {
		h.clientRooms[client.GetID()] = make(map[string]struct{})
	}
	h.clientRooms[client.GetID()][roomID] = struct{}{}

	_, hasSubscription := h.subscriptions[roomID]
	h.mu.Unlock()

	if !hasSubscription {
		if err := h.subscribeRoom(ctx, roomID); err != nil {
			return stackErr.Error(err)
		}
	}

	log.Infow("client joined room", "client_id", client.GetID(), "room_id", roomID)
	return nil
}

func (h *Hub) LeaveRoom(ctx context.Context, client IClient, roomID string) error {
	log := logging.FromContext(ctx)
	if client == nil {
		return stackErr.Error(errors.New("client is nil"))
	}
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return stackErr.Error(errors.New("room_id is required"))
	}

	shouldUnsubscribe := false

	h.mu.Lock()
	room, exists := h.rooms[roomID]
	if exists {
		room.RemoveClient(ctx, client)
		if room.IsEmpty() {
			delete(h.rooms, roomID)
			shouldUnsubscribe = true
		}
	}

	if rooms, ok := h.clientRooms[client.GetID()]; ok {
		delete(rooms, roomID)
		if len(rooms) == 0 {
			delete(h.clientRooms, client.GetID())
		}
	}
	h.mu.Unlock()

	if shouldUnsubscribe {
		h.unsubscribeRoom(ctx, roomID)
	}

	log.Infow("client left room", "client_id", client.GetID(), "room_id", roomID)
	return nil
}

func (h *Hub) HandleMessage(ctx context.Context, client IClient, msg Message) error {
	if client == nil {
		return stackErr.Error(errors.New("client is nil"))
	}
	if ctx == nil {
		ctx = context.Background()
	}

	switch msg.Action {
	case ActionJoinRoom:
		return h.JoinRoom(ctx, client, msg.RoomID)

	case ActionLeaveRoom:
		return h.LeaveRoom(ctx, client, msg.RoomID)

	case ActionChatMessage, ActionTyping, ActionSeen:
		roomID := strings.TrimSpace(msg.RoomID)
		if roomID == "" {
			return stackErr.Error(errors.New("room_id is required"))
		}
		if h.redisClient == nil {
			return stackErr.Error(errors.New("redis client is nil"))
		}
		if msg.SenderID == "" {
			msg.SenderID = client.GetUserID()
		}
		if msg.SenderID == "" {
			msg.SenderID = client.GetID()
		}

		payload, err := json.Marshal(msg)
		if err != nil {
			return stackErr.Error(fmt.Errorf("marshal websocket message: %v", err))
		}
		if err := h.redisClient.Publish(ctx, roomChannelName(roomID), payload).Err(); err != nil {
			return stackErr.Error(fmt.Errorf("publish redis message: %v", err))
		}
		return nil
	case ActionPresence:
		h.publishPresence(ctx, client.GetUserID(), "online")
		return nil
	}

	return stackErr.Error(fmt.Errorf("unsupported websocket action: %s", msg.Action))
}

func (h *Hub) Close(ctx context.Context) {
	log := logging.FromContext(ctx)
	h.closeMu.Do(func() {
		h.mu.Lock()
		h.isClosed = true
		clients := make([]IClient, 0, len(h.clients))
		for _, client := range h.clients {
			clients = append(clients, client)
		}
		subscriptions := make([]*roomSubscription, 0, len(h.subscriptions))
		for _, sub := range h.subscriptions {
			subscriptions = append(subscriptions, sub)
		}

		h.clients = make(map[string]IClient)
		h.rooms = make(map[string]IRoom)
		h.clientRooms = make(map[string]map[string]struct{})
		h.subscriptions = make(map[string]*roomSubscription)
		h.mu.Unlock()

		for _, sub := range subscriptions {
			if err := sub.Close(); err != nil {
				log.Warnw("failed to close redis pubsub", zap.Error(err))
			}
		}
		for _, client := range clients {
			client.Close(ctx)
		}

		log.Infow("hub closed")
	})
}

func (h *Hub) subscribeRoom(ctx context.Context, roomID string) error {
	log := logging.FromContext(ctx)
	if h.redisClient == nil {
		return stackErr.Error(errors.New("redis client is nil"))
	}

	h.mu.Lock()
	if h.isClosed {
		h.mu.Unlock()
		return stackErr.Error(errors.New("hub is closed"))
	}
	if _, exists := h.subscriptions[roomID]; exists {
		h.mu.Unlock()
		return nil
	}
	subCtx, cancel := context.WithCancel(ctx)
	pubsub := h.redisClient.Subscribe(subCtx, roomChannelName(roomID))
	sub := &roomSubscription{
		pubsub: pubsub,
		cancel: cancel,
	}
	h.subscriptions[roomID] = sub
	h.mu.Unlock()

	if _, err := pubsub.Receive(subCtx); err != nil {
		h.removeSubscription(subCtx, roomID, sub)
		_ = sub.Close()
		return stackErr.Error(fmt.Errorf("subscribe to redis room channel: %v", err))
	}

	go h.consumeRoomMessages(subCtx, roomID, sub)
	log.Infow("subscribed redis room channel", "room_id", roomID, "channel", roomChannelName(roomID))
	return nil
}

func (h *Hub) unsubscribeRoom(ctx context.Context, roomID string) {
	log := logging.FromContext(ctx)
	sub := h.detachSubscription(ctx, roomID)
	if sub == nil {
		return
	}
	if err := sub.Close(); err != nil {
		log.Warnw("failed to unsubscribe redis room channel", "room_id", roomID, zap.Error(err))
		return
	}
	log.Infow("unsubscribed redis room channel", "room_id", roomID, "channel", roomChannelName(roomID))
}

func (h *Hub) consumeRoomMessages(ctx context.Context, roomID string, sub *roomSubscription) {
	log := logging.FromContext(ctx)
	defer func() {
		h.removeSubscription(ctx, roomID, sub)
		if err := sub.Close(); err != nil {
			log.Debugw("error while closing redis pubsub from consumer", "room_id", roomID, zap.Error(err))
		}
	}()

	channel := sub.pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-channel:
			if !ok {
				return
			}
			h.broadcastLocal(ctx, roomID, []byte(message.Payload))
		}
	}
}

func (h *Hub) broadcastLocal(ctx context.Context, roomID string, payload []byte) {
	h.mu.RLock()
	room, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	room.Broadcast(ctx, payload)
}

func (h *Hub) detachSubscription(ctx context.Context, roomID string) *roomSubscription {
	h.mu.Lock()
	defer h.mu.Unlock()

	sub := h.subscriptions[roomID]
	delete(h.subscriptions, roomID)
	return sub
}

func (h *Hub) removeSubscription(ctx context.Context, roomID string, expected *roomSubscription) {
	h.mu.Lock()
	defer h.mu.Unlock()

	current, ok := h.subscriptions[roomID]
	if !ok {
		return
	}
	if expected != nil && current != expected {
		return
	}
	delete(h.subscriptions, roomID)
}

func roomChannelName(roomID string) string {
	return roomChannelPrefix + roomID
}

func (h *Hub) publishPresence(ctx context.Context, userID, status string) {
	if h.redisClient == nil || strings.TrimSpace(userID) == "" {
		return
	}
	key := "chat:presence:" + strings.TrimSpace(userID)
	if status == "online" {
		_ = h.redisClient.Set(ctx, key, "online", presenceTTL).Err()
	} else {
		_ = h.redisClient.Del(ctx, key).Err()
	}
	payload, err := json.Marshal(Message{
		Action:   ActionPresence,
		SenderID: userID,
		Data:     json.RawMessage(fmt.Sprintf(`{"status":%q}`, status)),
	})
	if err == nil {
		for _, roomID := range h.roomsForUser(userID) {
			_ = h.redisClient.Publish(ctx, roomChannelName(roomID), payload).Err()
		}
	}
}

func (h *Hub) roomsForUser(userID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	roomIDs := make([]string, 0)
	for _, client := range h.clients {
		if client.GetUserID() != userID {
			continue
		}
		for roomID := range h.clientRooms[client.GetID()] {
			roomIDs = append(roomIDs, roomID)
		}
	}
	return roomIDs
}
