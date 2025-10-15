package session

import (
	"context"
	"fmt"
	"time"

	"github.com/askwhyharsh/neartalk/pkg/logger"
)

type Manager struct {
	service *Service
	logger  logger.Logger
}

func NewManager(service *Service, log logger.Logger) *Manager {
	return &Manager{
		service: service,
		logger:  log,
	}
}

// Start begins background cleanup of inactive sessions
func (m *Manager) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	m.logger.Info("Session Manager started")

	for {
		select {
		case <-ticker.C:
			if err := m.cleanupInactiveSessions(ctx); err != nil {
				m.logger.Error("Failed to cleanup inactive sessions", "error", err)
			}
		case <-ctx.Done():
			m.logger.Info("Session Manager stopped")
			return
		}
	}
}

func (m *Manager) cleanupInactiveSessions(ctx context.Context) error {
	m.logger.Debug("Cleaning up inactive sessions")
	// Redis TTL handles this automatically, but we log it
	m.logger.Debug("Inactive sessions cleanup completed")
	return nil
}

// ValidateSession checks if a session exists and is valid
func (m *Manager) ValidateSession(ctx context.Context, sessionID string) error {
	exists, err := m.service.Exists(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to validate session: %w", err)
	}

	if !exists {
		return fmt.Errorf("session not found or expired")
	}

	// Update last seen
	if err := m.service.UpdateLastSeen(ctx, sessionID); err != nil {
		m.logger.Error("Failed to update last seen", "session_id", sessionID, "error", err)
	}

	return nil
}
