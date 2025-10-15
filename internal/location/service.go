package location

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/askwhyharsh/peoplearoundme/internal/storage"

)

type Service struct {
	redis            storage.RedisClient
	geohashPrecision int
	minRadius        int
	maxRadius        int
}

type Location struct {
	SessionID string    `json:"session_id"`
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Radius    int       `json:"radius"`
	Geohash   string    `json:"geohash"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NearbyUser struct {
	SessionID string `json:"session_id"`
	Username  string `json:"username"`
	Distance  int    `json:"distance"`
}

func NewService(redisClient storage.RedisClient, geohashPrecision, minRadius, maxRadius int) *Service {
	return &Service{
		redis:            redisClient,
		geohashPrecision: geohashPrecision,
		minRadius:        minRadius,
		maxRadius:        maxRadius,
	}
}

func (s *Service) UpdateLocation(ctx context.Context, sessionID string, lat, lon float64, radius int) error {
	// Validate radius
	if radius < s.minRadius || radius > s.maxRadius {
		return fmt.Errorf("radius must be between %d and %d meters", s.minRadius, s.maxRadius)
	}
	
	// Generate geohash
	geohash := Encode(lat, lon, s.geohashPrecision)
	
	location := &Location{
		SessionID: sessionID,
		Lat:       lat,
		Lon:       lon,
		Radius:    radius,
		Geohash:   geohash,
		UpdatedAt: time.Now(),
	}
	
	// Save to Redis
	key := s.locationKey(sessionID)
	data, err := json.Marshal(location)
	if err != nil {
		return fmt.Errorf("failed to marshal location: %w", err)
	}
	
	// Store location with 5 minute TTL (auto-refresh on activity)
	if err := s.redis.Set(ctx, key, data, 5*time.Minute); err != nil {
		return fmt.Errorf("failed to store location: %w", err)
	}
	
	// Add to geohash index
	geohashKey := s.geohashKey(geohash)
	if err := s.redis.SAdd(ctx, geohashKey, sessionID); err != nil {
		return fmt.Errorf("failed to add to geohash index: %w", err)
	}
	
	// Set expiration on geohash index
	s.redis.Expire(ctx, geohashKey, 5*time.Minute)
	
	return nil
}

func (s *Service) GetLocation(ctx context.Context, sessionID string) (*Location, error) {
	key := s.locationKey(sessionID)
	data, err := s.redis.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("location not found")
		}
		return nil, fmt.Errorf("failed to get location: %w", err)
	}
	
	var location Location
	if err := json.Unmarshal([]byte(data), &location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal location: %w", err)
	}
	
	return &location, nil
}

func (s *Service) GetNearbyUsers(ctx context.Context, sessionID string, getUsernameFn func(string) string) ([]NearbyUser, error) {
	userLoc, err := s.GetLocation(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	
	// Get geohashes to query (current + neighbors)
	geohashes := s.getGeohashesInRadius(userLoc.Geohash)
	
	// Collect candidates from all geohash cells
	candidateMap := make(map[string]bool)
	for _, gh := range geohashes {
		candidates, err := s.getUsersInGeohash(ctx, gh)
		if err != nil {
			continue
		}
		for _, c := range candidates {
			if c != sessionID {
				candidateMap[c] = true
			}
		}
	}
	
	// Calculate actual distances
	nearby := make([]NearbyUser, 0)
	for candidateID := range candidateMap {
		candidateLoc, err := s.GetLocation(ctx, candidateID)
		if err != nil {
			continue
		}
		
		distance := HaversineDistance(
			userLoc.Lat, userLoc.Lon,
			candidateLoc.Lat, candidateLoc.Lon,
		)
		
		// Check if within radius
		if distance <= float64(userLoc.Radius) {
			approxDist := RoundToNearest50(distance)
			nearby = append(nearby, NearbyUser{
				SessionID: candidateID,
				Username:  getUsernameFn(candidateID),
				Distance:  approxDist,
			})
		}
	}
	
	return nearby, nil
}

func (s *Service) GetGeohash(ctx context.Context, sessionID string) (string, error) {
	location, err := s.GetLocation(ctx, sessionID)
	if err != nil {
		return "", err
	}
	return location.Geohash, nil
}

func (s *Service) DeleteLocation(ctx context.Context, sessionID string) error {
	// Get current location to remove from geohash index
	location, err := s.GetLocation(ctx, sessionID)
	if err == nil {
		geohashKey := s.geohashKey(location.Geohash)
		s.redis.SRem(ctx, geohashKey, sessionID)
	}
	
	// Delete location
	key := s.locationKey(sessionID)
	return s.redis.Del(ctx, key)
}

func (s *Service) CleanupStaleLocations(ctx context.Context) error {
	// This is handled automatically by Redis TTL
	// But we can explicitly clean up geohash indices
	pattern := "geohash:*"
	iter := s.redis.Scan(ctx, 0, pattern, 100).Iterator()
	
	for iter.Next(ctx) {
		key := iter.Val()
		// Check if set is empty
		count, err := s.redis.SCard(ctx, key)
		if err != nil || count == 0 {
			s.redis.Del(ctx, key)
		}
	}
	
	return iter.Err()
}

func (s *Service) getGeohashesInRadius(geohash string) []string {
	geohashes := []string{geohash}
	neighbors := GetNeighbors(geohash)
	geohashes = append(geohashes, neighbors...)
	return geohashes
}

func (s *Service) getUsersInGeohash(ctx context.Context, geohash string) ([]string, error) {
	key := s.geohashKey(geohash)
	return s.redis.SMembers(ctx, key)
}

func (s *Service) locationKey(sessionID string) string {
	return fmt.Sprintf("location:%s", sessionID)
}

func (s *Service) geohashKey(geohash string) string {
	return fmt.Sprintf("geohash:%s", geohash)
}