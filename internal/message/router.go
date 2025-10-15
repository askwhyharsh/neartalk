package message

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/askwhyharsh/peoplearoundme/internal/storage"
)

type Router struct {
	redis storage.RedisClient
	store *Store
}

type BroadcastMessage struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Distance  string    `json:"distance,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func NewRouter(redisClient storage.RedisClient, store *Store) *Router {
	return &Router{
		redis: redisClient,
		store: store,
	}
}

func (r *Router) RouteMessage(ctx context.Context, msg *Message) error {
	// Save message to store
	if err := r.store.Save(ctx, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	
	// Publish to Redis pub/sub for real-time delivery
	channel := r.channelName(msg.Geohash)
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	if err := r.redis.Publish(ctx, channel, data); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	
	return nil
}

func (r *Router) Subscribe(ctx context.Context, geohash string, handler func(*Message)) error {
	channel := r.channelName(geohash)
	pubsub := r.redis.Subscribe(ctx, channel)
	defer pubsub.Close()
	
	ch := pubsub.Channel()
	
	for {
		select {
		case msg := <-ch:
			var message Message
			if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
				continue
			}
			handler(&message)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *Router) channelName(geohash string) string {
	return fmt.Sprintf("chat:%s", geohash)
}