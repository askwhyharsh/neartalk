package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type PostgresClient struct {
	db *sql.DB
}

type BanRecord struct {
	ID        int64
	SessionID string
	IPAddress string
	Reason    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type AnalyticsRecord struct {
	ID              int64
	Date            time.Time
	TotalSessions   int
	TotalMessages   int
	UniqueLocations int
	AvgRadius       float64
}

func NewPostgresClient(connStr string) (*PostgresClient, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	client := &PostgresClient{db: db}

	// Initialize schema
	if err := client.initSchema(); err != nil {
		return nil, err
	}

	return client, nil
}

func (p *PostgresClient) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS bans (
		id SERIAL PRIMARY KEY,
		session_id VARCHAR(255),
		ip_address VARCHAR(45),
		reason TEXT,
		expires_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_session_id (session_id),
		INDEX idx_ip_address (ip_address),
		INDEX idx_expires_at (expires_at)
	);

	CREATE TABLE IF NOT EXISTS analytics (
		id SERIAL PRIMARY KEY,
		date DATE UNIQUE,
		total_sessions INT DEFAULT 0,
		total_messages INT DEFAULT 0,
		unique_locations INT DEFAULT 0,
		avg_radius FLOAT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := p.db.Exec(schema)
	return err
}

func (p *PostgresClient) Close() error {
	return p.db.Close()
}

// Ban operations
func (p *PostgresClient) AddBan(ctx context.Context, sessionID, ipAddress, reason string, duration time.Duration) error {
	query := `
		INSERT INTO bans (session_id, ip_address, reason, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	expiresAt := time.Now().Add(duration)
	_, err := p.db.ExecContext(ctx, query, sessionID, ipAddress, reason, expiresAt)
	return err
}

func (p *PostgresClient) IsBanned(ctx context.Context, sessionID, ipAddress string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM bans
		WHERE (session_id = $1 OR ip_address = $2)
		AND expires_at > NOW()
	`

	var count int
	err := p.db.QueryRowContext(ctx, query, sessionID, ipAddress).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *PostgresClient) GetBanReason(ctx context.Context, sessionID, ipAddress string) (string, error) {
	query := `
		SELECT reason FROM bans
		WHERE (session_id = $1 OR ip_address = $2)
		AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	var reason string
	err := p.db.QueryRowContext(ctx, query, sessionID, ipAddress).Scan(&reason)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return reason, nil
}

func (p *PostgresClient) CleanupExpiredBans(ctx context.Context) error {
	query := `DELETE FROM bans WHERE expires_at <= NOW()`
	_, err := p.db.ExecContext(ctx, query)
	return err
}

// Analytics operations
func (p *PostgresClient) RecordDailyStats(ctx context.Context, date time.Time, sessions, messages, locations int, avgRadius float64) error {
	query := `
		INSERT INTO analytics (date, total_sessions, total_messages, unique_locations, avg_radius)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (date) DO UPDATE SET
			total_sessions = analytics.total_sessions + EXCLUDED.total_sessions,
			total_messages = analytics.total_messages + EXCLUDED.total_messages,
			unique_locations = EXCLUDED.unique_locations,
			avg_radius = EXCLUDED.avg_radius
	`

	_, err := p.db.ExecContext(ctx, query, date, sessions, messages, locations, avgRadius)
	return err
}

func (p *PostgresClient) GetAnalytics(ctx context.Context, startDate, endDate time.Time) ([]AnalyticsRecord, error) {
	query := `
		SELECT id, date, total_sessions, total_messages, unique_locations, avg_radius
		FROM analytics
		WHERE date BETWEEN $1 AND $2
		ORDER BY date DESC
	`

	rows, err := p.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []AnalyticsRecord
	for rows.Next() {
		var record AnalyticsRecord
		if err := rows.Scan(
			&record.ID,
			&record.Date,
			&record.TotalSessions,
			&record.TotalMessages,
			&record.UniqueLocations,
			&record.AvgRadius,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func (p *PostgresClient) GetTotalStats(ctx context.Context) (*AnalyticsRecord, error) {
	query := `
		SELECT
			SUM(total_sessions) as total_sessions,
			SUM(total_messages) as total_messages,
			AVG(unique_locations) as unique_locations,
			AVG(avg_radius) as avg_radius
		FROM analytics
	`

	var record AnalyticsRecord
	err := p.db.QueryRowContext(ctx, query).Scan(
		&record.TotalSessions,
		&record.TotalMessages,
		&record.UniqueLocations,
		&record.AvgRadius,
	)

	if err != nil {
		return nil, err
	}

	return &record, nil
}