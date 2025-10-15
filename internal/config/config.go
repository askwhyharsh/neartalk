package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env          string
	Server      ServerConfig
	Redis       RedisConfig
	RateLimit   RateLimitConfig
	Session     SessionConfig
	Spam        SpamConfig
	Location    LocationConfig
	Monitoring  MonitoringConfig
}

type ServerConfig struct {
	Port string
	Host string
	Env  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type RateLimitConfig struct {
	MessagesPerMin       int
	LocationPerMin       int
	MaxUsernameChanges   int
	SessionsPerIPPerHour int
}

type SessionConfig struct {
	TTL        time.Duration
	MessageTTL time.Duration
}

type SpamConfig struct {
	ProfanityEnabled       bool
	DuplicateWindowSeconds int
	MaxURLsPerMessage      int
}

type LocationConfig struct {
	GeohashPrecision int
	MinRadiusMeters  int
	MaxRadiusMeters  int
}

type MonitoringConfig struct {
	EnableMetrics bool
	LogLevel      string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "0.0.0.0"),
			Env:  getEnv("ENV", "development"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		RateLimit: RateLimitConfig{
			MessagesPerMin:       getEnvInt("RATE_LIMIT_MESSAGES_PER_MIN", 10),
			LocationPerMin:       getEnvInt("RATE_LIMIT_LOCATION_PER_MIN", 6),
			MaxUsernameChanges:   getEnvInt("RATE_LIMIT_MAX_USERNAME_CHANGES", 3),
			SessionsPerIPPerHour: getEnvInt("RATE_LIMIT_SESSIONS_PER_IP_PER_HOUR", 10),
		},
		Session: SessionConfig{
			TTL:        time.Duration(getEnvInt("SESSION_TTL_MINUTES", 30)) * time.Minute,
			MessageTTL: time.Duration(getEnvInt("MESSAGE_TTL_MINUTES", 30)) * time.Minute,
		},
		Spam: SpamConfig{
			ProfanityEnabled:       getEnvBool("SPAM_PROFANITY_ENABLED", true),
			DuplicateWindowSeconds: getEnvInt("SPAM_DUPLICATE_WINDOW_SECONDS", 30),
			MaxURLsPerMessage:      getEnvInt("SPAM_MAX_URLS_PER_MESSAGE", 2),
		},
		Location: LocationConfig{
			GeohashPrecision: getEnvInt("GEOHASH_PRECISION", 7),
			MinRadiusMeters:  getEnvInt("MIN_RADIUS_METERS", 100),
			MaxRadiusMeters:  getEnvInt("MAX_RADIUS_METERS", 2000),
		},
		Monitoring: MonitoringConfig{
			EnableMetrics: getEnvBool("ENABLE_METRICS", true),
			LogLevel:      getEnv("LOG_LEVEL", "info"),
		},
	}

	return cfg, nil
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

func (c *Config) ServerAddr() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}