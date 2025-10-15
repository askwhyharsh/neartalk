package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, validate origin properly
	},
}

type Handler struct {
	hub           *Hub
	redis         *redis.Client
	sessionGetter SessionGetter
	locationGetter LocationGetter
	spamDetector  SpamDetector
	rateLimiter   RateLimiter
	messageTTL    time.Duration
}

type SessionGetter interface {
	Get(ctx context.Context, sessionID string) (*SessionData, error)
}

type LocationGetter interface {
	GetGeohash(ctx context.Context, sessionID string) (string, int, error)
}

type SpamDetector interface {
	ValidateMessage(ctx context.Context, sessionID, content string) error
	IncrementViolation(ctx context.Context, sessionID, violationType string) error
}

type RateLimiter interface {
	AllowMessage(ctx context.Context, sessionID string) (bool, error)
}

type SessionData struct {
	ID       string
	Username string
}

func NewHandler(hub *Hub, redis *redis.Client, sessionGetter SessionGetter, locationGetter LocationGetter, spamDetector SpamDetector, rateLimiter RateLimiter, messageTTL time.Duration) *Handler {
	return &Handler{
		hub:            hub,
		redis:          redis,
		sessionGetter:  sessionGetter,
		locationGetter: locationGetter,
		spamDetector:   spamDetector,
		rateLimiter:    rateLimiter,
		messageTTL:     messageTTL,
	}
}

func (h *Handler) HandleWebSocket(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}

	ctx := c.Request.Context()

	// Get session data
	session, err := h.sessionGetter.Get(ctx, sessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	// Get location data
	geohash, radius, err := h.locationGetter.GetGeohash(ctx, sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "location not set"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create client
	client := NewClient(h.hub, conn, sessionID, session.Username, geohash, radius)

	// Register client
	h.hub.register <- client

	// Start goroutines
	go client.WritePump()
	go h.handleClientMessages(client)

	client.ReadPump()
}

func (h *Handler) handleClientMessages(client *Client) {
	for {
		select {
		case <-client.ctx.Done():
			return
		default:
			_, messageData, err := client.conn.ReadMessage()
			if err != nil {
				return
			}

			var incoming IncomingMessage
			if err := json.Unmarshal(messageData, &incoming); err != nil {
				client.SendError("Invalid message format", "INVALID_FORMAT")
				continue
			}

			switch incoming.Type {
			case MessageTypeChat:
				h.handleChatMessage(client, &incoming)
			case MessageTypePing:
				client.send <- &Message{
					Type:      MessageTypePong,
					Timestamp: time.Now().Unix(),
				}
			}
		}
	}
}

func (h *Handler) handleChatMessage(client *Client, incoming *IncomingMessage) {
	ctx := context.Background()

	// Rate limiting
	allowed, err := h.rateLimiter.AllowMessage(ctx, client.sessionID)
	if err != nil || !allowed {
		client.SendError("Rate limit exceeded", "RATE_LIMIT")
		return
	}

	// Spam detection
	if err := h.spamDetector.ValidateMessage(ctx, client.sessionID, incoming.Content); err != nil {
		client.SendError(err.Error(), "SPAM_DETECTED")
		h.spamDetector.IncrementViolation(ctx, client.sessionID, "spam")
		return
	}

	// Create message
	message := NewChatMessage(
		client.sessionID,
		client.username,
		incoming.Content,
		client.geohash,
		"", // Distance will be calculated per recipient
	)

	// Store message in Redis
	if err := h.storeMessage(ctx, message); err != nil {
		log.Printf("Failed to store message: %v", err)
		client.SendError("Failed to send message", "INTERNAL_ERROR")
		return
	}

	// Broadcast to hub
	h.hub.broadcast <- message
}

func (h *Handler) storeMessage(ctx context.Context, msg *Message) error {
	key := fmt.Sprintf("messages:%s", msg.Geohash)
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Add to sorted set with timestamp as score
	if err := h.redis.ZAdd(ctx, key, redis.Z{
		Score:  float64(msg.Timestamp),
		Member: data,
	}).Err(); err != nil {
		return err
	}

	// Set expiration
	return h.redis.Expire(ctx, key, h.messageTTL).Err()
}

func (h *Handler) GetRecentMessages(ctx context.Context, geohash string, limit int64) ([]*Message, error) {
	key := fmt.Sprintf("messages:%s", geohash)

	// Get recent messages
	results, err := h.redis.ZRevRange(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}

	messages := make([]*Message, 0, len(results))
	for _, result := range results {
		var msg Message
		if err := json.Unmarshal([]byte(result), &msg); err != nil {
			continue
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}