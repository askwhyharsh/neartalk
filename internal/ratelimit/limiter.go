package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/askwhyharsh/neartalk/internal/config"
	"github.com/askwhyharsh/neartalk/internal/storage"
	"github.com/redis/go-redis/v9"
)

// RateLimiter defines the contract for enforcing and managing rate limits.
type RateLimiter interface {
	// AllowMessage checks if a session is allowed to send a message right now.
	AllowMessage(ctx context.Context, sessionID string) (bool, error)

	// AllowLocationUpdate checks if a session can update its location.
	AllowLocationUpdate(ctx context.Context, sessionID string) (bool, error)

	// AllowUsernameChange checks if a session can change its username.
	// Returns (allowed, remaining_changes, error).
	AllowUsernameChange(ctx context.Context, sessionID string) (bool, int, error)

	// AllowSessionCreation checks if an IP can create a new session.
	AllowSessionCreation(ctx context.Context, ip string) (bool, error)

	// AllowIPRequest checks if an IP can make a request.
	AllowIPRequest(ctx context.Context, ip string) (bool, error)

	// GetRemainingMessages returns how many messages a session can still send in the current window.
	GetRemainingMessages(ctx context.Context, sessionID string) (int, error)

	// ResetLimits clears all rate limit counters for a session.
	ResetLimits(ctx context.Context, sessionID string) error
}

type Limiter struct {
	redis  storage.RedisClient
	config config.RateLimitConfig
}

func NewLimiter(redisClient storage.RedisClient, config config.RateLimitConfig) *Limiter {
	return &Limiter{
		redis:  redisClient,
		config: config,
	}
}

// AllowMessage checks if a session can send a message
func (l *Limiter) AllowMessage(ctx context.Context, sessionID string) (bool, error) {
	key := fmt.Sprintf("ratelimit:msg:%s", sessionID)
	return l.checkSlidingWindow(ctx, key, l.config.MessagesPerMin, 60)
}

// AllowLocationUpdate checks if a session can update location
func (l *Limiter) AllowLocationUpdate(ctx context.Context, sessionID string) (bool, error) {
	key := fmt.Sprintf("ratelimit:location:%s", sessionID)
	return l.checkSlidingWindow(ctx, key, l.config.LocationUpdatesPerMin, 60)
}

// AllowUsernameChange checks if a session can change username
func (l *Limiter) AllowUsernameChange(ctx context.Context, sessionID string) (bool, int, error) {
	key := fmt.Sprintf("ratelimit:username:%s", sessionID)

	count, err := l.redis.Incr(ctx, key)
	if err != nil {
		return false, 0, fmt.Errorf("failed to check username rate limit: %w", err)
	}

	// Set expiration on first increment (24 hours)
	if count == 1 {
		l.redis.Expire(ctx, key, 24*time.Hour)
	}

	remaining := l.config.MaxUsernameChanges - int(count) + 1
	if remaining < 0 {
		remaining = 0
	}

	return count <= int64(l.config.MaxUsernameChanges), remaining, nil
}

// AllowSessionCreation checks if an IP can create a new session
func (l *Limiter) AllowSessionCreation(ctx context.Context, ip string) (bool, error) {
	key := fmt.Sprintf("ratelimit:ip:%s:sessions", ip)

	count, err := l.redis.Incr(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check session creation rate limit: %w", err)
	}

	// Set expiration on first increment (1 hour)
	if count == 1 {
		l.redis.Expire(ctx, key, time.Hour)
	}

	return count <= int64(l.config.SessionsPerIPPerHour), nil
}

// AllowIPRequest checks if an IP can make a request
func (l *Limiter) AllowIPRequest(ctx context.Context, ip string) (bool, error) {
	key := fmt.Sprintf("ratelimit:ip:%s:requests", ip)
	return l.checkSlidingWindow(ctx, key, l.config.RequestsPerMinute, 60)
}

// checkSlidingWindow implements a sliding window rate limiter using sorted sets
func (l *Limiter) checkSlidingWindow(ctx context.Context, key string, maxCount int, windowSec int) (bool, error) {
	now := time.Now().Unix()
	windowStart := now - int64(windowSec)

	// Remove old entries outside the window
	if err := l.redis.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart)); err != nil {
		return false, fmt.Errorf("failed to clean old entries: %w", err)
	}

	// Count entries in current window
	count, err := l.redis.ZCard(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to count entries: %w", err)
	}

	if count >= int64(maxCount) {
		return false, nil
	}

	// Add new entry
	if err := l.redis.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d", now),
	}); err != nil {
		return false, fmt.Errorf("failed to add entry: %w", err)
	}

	// Set expiration
	l.redis.Expire(ctx, key, time.Duration(windowSec)*time.Second)

	return true, nil
}

// GetRemainingMessages returns how many messages a session can still send
func (l *Limiter) GetRemainingMessages(ctx context.Context, sessionID string) (int, error) {
	key := fmt.Sprintf("ratelimit:msg:%s", sessionID)
	count, err := l.redis.ZCard(ctx, key)
	if err != nil {
		return l.config.MessagesPerMin, nil
	}

	remaining := l.config.MessagesPerMin - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// ResetLimits resets all rate limits for a session (use with caution)
func (l *Limiter) ResetLimits(ctx context.Context, sessionID string) error {
	keys := []string{
		fmt.Sprintf("ratelimit:msg:%s", sessionID),
		fmt.Sprintf("ratelimit:location:%s", sessionID),
		fmt.Sprintf("ratelimit:username:%s", sessionID),
	}

	for _, key := range keys {
		if err := l.redis.Del(ctx, key); err != nil {
			return err
		}
	}

	return nil
}
