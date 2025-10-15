package message

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/askwhyharsh/neartalk/internal/storage"
	// "github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	redis storage.RedisClient
	ttl   time.Duration
}

type Message struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"sender_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Geohash   string    `json:"geohash"`
	Timestamp time.Time `json:"timestamp"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewStore(redisClient storage.RedisClient, ttl time.Duration) *Store {
	return &Store{
		redis: redisClient,
		ttl:   ttl,
	}
}

func (s *Store) Save(ctx context.Context, msg *Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	if msg.ExpiresAt.IsZero() {
		msg.ExpiresAt = msg.Timestamp.Add(s.ttl)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	key := s.messageKey(msg.Geohash)
	score := float64(msg.Timestamp.Unix())

	// Add to sorted set with timestamp as score
	if err := s.redis.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: data,
	}); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Set expiration on the sorted set
	s.redis.Expire(ctx, key, s.ttl)

	return nil
}

func (s *Store) GetRecent(ctx context.Context, geohash string, limit int) ([]*Message, error) {
	key := s.messageKey(geohash)

	// Get recent messages (sorted by timestamp descending)
	results, err := s.redis.ZRevRangeByScore(ctx, key, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   "+inf",
		Count: int64(limit),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]*Message, 0, len(results))
	now := time.Now()

	for _, data := range results {
		var msg Message
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			continue
		}

		// Skip expired messages
		if now.After(msg.ExpiresAt) {
			continue
		}

		messages = append(messages, &msg)
	}

	return messages, nil
}

func (s *Store) CleanupExpired(ctx context.Context) error {
	pattern := "messages:*"
	iter := s.redis.Scan(ctx, 0, pattern, 100).Iterator()

	now := time.Now().Unix()

	for iter.Next(ctx) {
		key := iter.Val()

		// Remove expired messages (score < current timestamp - TTL)
		expiredBefore := now - int64(s.ttl.Seconds())
		if err := s.redis.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", expiredBefore)); err != nil {
			continue
		}

		// Delete empty sorted sets
		count, err := s.redis.ZCard(ctx, key)
		if err == nil && count == 0 {
			s.redis.Del(ctx, key)
		}
	}

	return iter.Err()
}

func (s *Store) messageKey(geohash string) string {
	return fmt.Sprintf("messages:%s", geohash)
}
