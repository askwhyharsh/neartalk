package ratelimit

type Config struct {
	MessagesPerMin         int
	LocationUpdatesPerMin  int
	MaxUsernameChanges     int
	SessionsPerIPPerHour   int
	RequestsPerMinute      int
	ConcurrentConnections  int
}

func DefaultConfig() *Config {
	return &Config{
		MessagesPerMin:        10,
		LocationUpdatesPerMin: 6,
		MaxUsernameChanges:    3,
		SessionsPerIPPerHour:  10,
		RequestsPerMinute:     100,
		ConcurrentConnections: 5,
	}
}