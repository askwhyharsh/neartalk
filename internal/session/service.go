package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/askwhyharsh/neartalk/internal/storage"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SessionService interface {
	Create(ctx context.Context, ipAddress string) (*Session, error)
	Get(ctx context.Context, sessionID string) (*Session, error)
	UpdateUsername(ctx context.Context, sessionID, newUsername string) error
	UpdateLastSeen(ctx context.Context, sessionID string) error
	Delete(ctx context.Context, sessionID string) error
	GetRemainingChanges(ctx context.Context, sessionID string) (int, error)
	Exists(ctx context.Context, sessionID string) (bool, error)
}

type Service struct {
	redis      storage.RedisClient
	ttl        time.Duration
	maxChanges int
}

type Session struct {
	ID                  string    `json:"id"`
	Username            string    `json:"username"`
	UsernameChangeCount int       `json:"username_change_count"`
	MaxUsernameChanges  int       `json:"max_username_changes"`
	CreatedAt           time.Time `json:"created_at"`
	LastSeen            time.Time `json:"last_seen"`
	IPAddress           string    `json:"ip_address"`
}

func NewService(redisClient storage.RedisClient, ttl time.Duration, maxChanges int) *Service {
	return &Service{
		redis:      redisClient,
		ttl:        ttl,
		maxChanges: maxChanges,
	}
}

func (s *Service) Create(ctx context.Context, ipAddress string) (*Session, error) {
	session := &Session{
		ID:                  uuid.New().String(),
		Username:            generateRandomUsername(),
		UsernameChangeCount: 0,
		MaxUsernameChanges:  s.maxChanges,
		CreatedAt:           time.Now(),
		LastSeen:            time.Now(),
		IPAddress:           ipAddress,
	}

	if err := s.save(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

func (s *Service) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := s.sessionKey(sessionID)
	data, err := s.redis.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (s *Service) UpdateUsername(ctx context.Context, sessionID, newUsername string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.UsernameChangeCount >= session.MaxUsernameChanges {
		return fmt.Errorf("username change limit reached")
	}

	session.Username = newUsername
	session.UsernameChangeCount++
	session.LastSeen = time.Now()

	return s.save(ctx, session)
}

func (s *Service) UpdateLastSeen(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.LastSeen = time.Now()
	return s.save(ctx, session)
}

func (s *Service) Delete(ctx context.Context, sessionID string) error {
	key := s.sessionKey(sessionID)
	return s.redis.Del(ctx, key)
}

func (s *Service) save(ctx context.Context, session *Session) error {
	key := s.sessionKey(session.ID)
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return s.redis.Set(ctx, key, data, s.ttl)
}

func (s *Service) sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

func (s *Service) GetRemainingChanges(ctx context.Context, sessionID string) (int, error) {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return 0, err
	}

	remaining := session.MaxUsernameChanges - session.UsernameChangeCount
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

func (s *Service) Exists(ctx context.Context, sessionID string) (bool, error) {
	key := s.sessionKey(sessionID)
	count, err := s.redis.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
