package websocket

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/askwhyharsh/neartalk/internal/storage"
)

type Hub struct {
	clients    map[string]*Client
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	redis      storage.RedisClient
	mu         sync.RWMutex
	ctx        context.Context
}

func NewHub(ctx context.Context, redisClient storage.RedisClient) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		redis:      redisClient,
		ctx:        ctx,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastMessage(message)
		case <-h.ctx.Done():
			h.shutdown()
			return
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.sessionID] = client

	// Store in Redis for distributed tracking
	key := "ws:active"
	h.redis.SAdd(h.ctx, key, client.sessionID)

	// Notify others about new user
	h.broadcastUserJoined(client)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.sessionID]; ok {
		delete(h.clients, client.sessionID)
		close(client.send)

		// Remove from Redis
		key := "ws:active"
		h.redis.SRem(h.ctx, key, client.sessionID)

		// Notify others about user leaving
		h.broadcastUserLeft(client)
	}
}

func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Publish to Redis for multi-server support
	channel := "chat:" + message.Geohash
	data, _ := json.Marshal(message)
	h.redis.Publish(h.ctx, channel, data)

	// Broadcast to local clients
	for _, client := range h.clients {
		// Only send to clients in the same geohash or nearby
		if client.shouldReceiveMessage(message) {
			select {
			case client.send <- message:
			default:
				// Client's send channel is full, close it
				close(client.send)
				delete(h.clients, client.sessionID)
			}
		}
	}
}

func (h *Hub) broadcastUserJoined(client *Client) {
	message := &Message{
		Type:      MessageTypeUserJoined,
		Username:  client.username,
		UserCount: h.getUserCount(),
	}

	for _, c := range h.clients {
		if c.sessionID != client.sessionID {
			select {
			case c.send <- message:
			default:
			}
		}
	}
}

func (h *Hub) broadcastUserLeft(client *Client) {
	message := &Message{
		Type:      MessageTypeUserLeft,
		Username:  client.username,
		UserCount: h.getUserCount(),
	}

	for _, c := range h.clients {
		select {
		case c.send <- message:
		default:
		}
	}
}

func (h *Hub) getUserCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) GetClient(sessionID string) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, ok := h.clients[sessionID]
	return client, ok
}

func (h *Hub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		close(client.send)
	}
	h.clients = make(map[string]*Client)
}

func (h *Hub) BroadcastToGeohash(geohash string, message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if client.geohash == geohash || isNeighborGeohash(client.geohash, geohash) {
			select {
			case client.send <- message:
			default:
			}
		}
	}
}

// Helper function to check if two geohashes are neighbors
func isNeighborGeohash(gh1, gh2 string) bool {
	if len(gh1) < 4 || len(gh2) < 4 {
		return false
	}
	// Simple check: first 4 characters should be similar for proximity
	return gh1[:4] == gh2[:4]
}
