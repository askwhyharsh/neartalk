package ratelimit

import "github.com/askwhyharsh/peoplearoundme/internal/config"

func DefaultConfig() *config.RateLimitConfig {
	return &config.RateLimitConfig{
		MessagesPerMin:        10,
		LocationUpdatesPerMin: 6,
		MaxUsernameChanges:    3,
		SessionsPerIPPerHour:  10,
		RequestsPerMinute:     100,
		ConcurrentConnections: 5,
	}
}