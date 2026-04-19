package socket

import (
	"context"
	"sync"
)

type room struct {
	mu      sync.RWMutex
	id      string
	clients map[string]*client
}

func newRoom(roomID string) *room {
	return &room{
		id:      roomID,
		clients: make(map[string]*client),
	}
}

func (r *room) addClient(c *client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[c.id] = c
}

func (r *room) removeClient(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, clientID)
}

func (r *room) broadcast(ctx context.Context, payload []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		if c != nil {
			c.send(ctx, payload)
		}
	}
}

func (r *room) isEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients) == 0
}
