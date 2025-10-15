package message

import (
	"context"
	"time"

	"github.com/askwhyharsh/peoplearoundme/pkg/logger"
)

type TTLManager struct {
	store  *Store
	logger logger.Logger
}

func NewTTLManager(store *Store, log logger.Logger) *TTLManager {
	return &TTLManager{
		store:  store,
		logger: log,
	}
}

func (m *TTLManager) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	m.logger.Info("TTL Manager started")
	
	for {
		select {
		case <-ticker.C:
			if err := m.cleanupExpiredMessages(ctx); err != nil {
				m.logger.Error("Failed to cleanup expired messages", "error", err)
			}
		case <-ctx.Done():
			m.logger.Info("TTL Manager stopped")
			return
		}
	}
}

func (m *TTLManager) cleanupExpiredMessages(ctx context.Context) error {
	m.logger.Debug("Cleaning up expired messages")
	
	if err := m.store.CleanupExpired(ctx); err != nil {
		return err
	}
	
	m.logger.Debug("Expired messages cleanup completed")
	return nil
}