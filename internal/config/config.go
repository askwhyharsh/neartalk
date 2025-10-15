package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Redis    RedisConfig
	RateLimit RateLimitConfig
	Session  SessionConfig
	Spam     SpamConfig
	Location LocationConfig
	Monitoring MonitoringConfig
}

type ServerConfig struct {
	Port string
	Env  string
	Host string
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
	TTLMinutes     int
	MessageTTLMinutes int
}

type SpamConfig struct {
	ProfanityEnabled      bool
	DuplicateWindowSeconds int
	MaxURLsPerMessage     int
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
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
			Host: getEnv("HOST", "0.0.0.0"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		RateLimit: RateLimitConfig{
			MessagesPerMin:       getEnvAsInt("RATE_LIMIT_MESSAGES_PER_MIN", 10),
			LocationPerMin:       getEnvAsInt("RATE_LIMIT_LOCATION_PER_MIN", 6),
			MaxUsernameChanges:   getEnvAsInt("RATE_LIMIT_MAX_USERNAME_CHANGES", 3),
			SessionsPerIPPerHour: getEnvAsInt("RATE_LIMIT_SESSIONS_PER_IP_PER_HOUR", 10),
		},
		Session: SessionConfig{
			TTLMinutes:        getEnvAsInt("SESSION_TTL_MINUTES", 30),
			MessageTTLMinutes: getEnvAsInt("MESSAGE_TTL_MINUTES", 30),
		},
		Spam: SpamConfig{
			ProfanityEnabled:      getEnvAsBool("SPAM_PROFANITY_ENABLED", true),
			DuplicateWindowSeconds: getEnvAsInt("SPAM_DUPLICATE_WINDOW_SECONDS", 30),
			MaxURLsPerMessage:     getEnvAsInt("SPAM_MAX_URLS_PER_MESSAGE", 2),
		},
		Location: LocationConfig{
			GeohashPrecision: getEnvAsInt("GEOHASH_PRECISION", 7),
			MinRadiusMeters:  getEnvAsInt("MIN_RADIUS_METERS", 100),
			MaxRadiusMeters:  getEnvAsInt("MAX_RADIUS_METERS", 2000),
		},
		Monitoring: MonitoringConfig{
			EnableMetrics: getEnvAsBool("ENABLE_METRICS", true),
			LogLevel:      getEnv("LOG_LEVEL", "info"),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}
