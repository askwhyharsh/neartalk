package spam

import (
	"context"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Detector struct {
	redis                  *redis.Client
	profanityEnabled       bool
	duplicateWindowSeconds int
	maxURLsPerMessage      int
	profanityWords         []string
	urlRegex               *regexp.Regexp
	mu                     sync.RWMutex
}

func NewDetector(redisClient *redis.Client, profanityEnabled bool, duplicateWindow, maxURLs int) *Detector {
	return &Detector{
		redis:                  redisClient,
		profanityEnabled:       profanityEnabled,
		duplicateWindowSeconds: duplicateWindow,
		maxURLsPerMessage:      maxURLs,
		profanityWords:         loadProfanityList(),
		urlRegex:               regexp.MustCompile(`https?://[^\s]+`),
	}
}

func (d *Detector) ValidateMessage(ctx context.Context, sessionID, content string) error {
	// Check message length
	if len(content) < 1 {
		return fmt.Errorf("message too short")
	}
	if len(content) > 500 {
		return fmt.Errorf("message too long (max 500 characters)")
	}

	// Check if empty after trimming
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("message cannot be empty")
	}

	// Check profanity
	if d.profanityEnabled {
		if d.containsProfanity(content) {
			return fmt.Errorf("message contains profanity")
		}
	}

	// Check for excessive URLs
	if d.hasExcessiveURLs(content) {
		return fmt.Errorf("too many URLs in message (max %d)", d.maxURLsPerMessage)
	}

	// Check for duplicate spam
	if err := d.checkDuplicateSpam(ctx, sessionID, content); err != nil {
		return err
	}

	return nil
}

func (d *Detector) containsProfanity(content string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	lowerContent := strings.ToLower(content)
	for _, word := range d.profanityWords {
		if strings.Contains(lowerContent, strings.ToLower(word)) {
			return true
		}
	}
	return false
}

func (d *Detector) hasExcessiveURLs(content string) bool {
	urls := d.urlRegex.FindAllString(content, -1)
	return len(urls) > d.maxURLsPerMessage
}

func (d *Detector) checkDuplicateSpam(ctx context.Context, sessionID, content string) error {
	// Create hash of the message content
	hash := fmt.Sprintf("%x", md5.Sum([]byte(content)))
	key := fmt.Sprintf("spam:msg:%s:%s", sessionID, hash)

	// Check if this exact message was sent recently
	exists, err := d.redis.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check duplicate: %w", err)
	}

	if exists > 0 {
		return fmt.Errorf("duplicate message detected (sent within %d seconds)", d.duplicateWindowSeconds)
	}

	// Store the message hash with TTL
	ttl := time.Duration(d.duplicateWindowSeconds) * time.Second
	if err := d.redis.Set(ctx, key, 1, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store message hash: %w", err)
	}

	return nil
}

func (d *Detector) IncrementViolation(ctx context.Context, sessionID string, violationType string) error {
	key := fmt.Sprintf("spam:violations:%s", sessionID)
	
	// Increment violation count
	if err := d.redis.HIncrBy(ctx, key, violationType, 1).Err(); err != nil {
		return err
	}

	// Set expiration (24 hours)
	return d.redis.Expire(ctx, key, 24*time.Hour).Err()
}

func (d *Detector) GetViolationCount(ctx context.Context, sessionID string) (map[string]int64, error) {
	key := fmt.Sprintf("spam:violations:%s", sessionID)
	violations, err := d.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for k, v := range violations {
		var count int64
		fmt.Sscanf(v, "%d", &count)
		result[k] = count
	}

	return result, nil
}

func (d *Detector) ShouldBan(ctx context.Context, sessionID string) (bool, string, error) {
	violations, err := d.GetViolationCount(ctx, sessionID)
	if err != nil {
		return false, "", err
	}

	// Ban rules
	if violations["profanity"] >= 3 {
		return true, "excessive profanity", nil
	}
	if violations["spam"] >= 5 {
		return true, "excessive spam", nil
	}

	totalViolations := int64(0)
	for _, count := range violations {
		totalViolations += count
	}

	if totalViolations >= 10 {
		return true, "excessive violations", nil
	}

	return false, "", nil
}