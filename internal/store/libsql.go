package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/tursodatabase/go-libsql"
)

type SQLStore struct {
	db *sql.DB
}

// Local sqlite: "file:./data/leetrss.db" or ":memory:"
// TursoDB: "libsql://[db-name]-[org].turso.io?authToken=..."
func NewSQLStore(dsn string) (*SQLStore, error) {
	db, err := sql.Open("libsql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// enable foreign keys for SQLite
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		// Ignore error for remote TursoDB (may not support PRAGMA)
		_ = err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &SQLStore{db: db}, nil
}

func (s *SQLStore) Close() error {
	return s.db.Close()
}

// returns the database connection for migrations and tests
func (s *SQLStore) DB() *sql.DB {
	return s.db
}

func (s *SQLStore) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, email, auth_provider, provider_subject, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.AuthProvider,
		user.ProviderSubject,
		user.CreatedAt.Format(time.RFC3339),
		user.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *SQLStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, auth_provider, provider_subject, created_at, updated_at
		FROM users WHERE email = ?
	`
	return s.scanUser(s.db.QueryRowContext(ctx, query, email))
}

func (s *SQLStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, auth_provider, provider_subject, created_at, updated_at
		FROM users WHERE id = ?
	`
	return s.scanUser(s.db.QueryRowContext(ctx, query, id))
}

func (s *SQLStore) GetUserByProvider(ctx context.Context, provider, subject string) (*User, error) {
	query := `
		SELECT id, email, auth_provider, provider_subject, created_at, updated_at
		FROM users WHERE auth_provider = ? AND provider_subject = ?
	`
	return s.scanUser(s.db.QueryRowContext(ctx, query, provider, subject))
}

func (s *SQLStore) scanUser(row *sql.Row) (*User, error) {
	var user User
	var createdAt, updatedAt string
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.AuthProvider,
		&user.ProviderSubject,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &user, nil
}

// --- Feed operations ---

func (s *SQLStore) CreateFeed(ctx context.Context, feed *Feed) error {
	usernamesJSON, err := json.Marshal(feed.Usernames)
	if err != nil {
		return fmt.Errorf("marshal usernames: %w", err)
	}

	query := `
		INSERT INTO feeds (id, user_id, name, secret, usernames, first_per_user, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.ExecContext(ctx, query,
		feed.ID,
		feed.UserID,
		feed.Name,
		feed.Secret,
		string(usernamesJSON),
		feed.FirstPerUser,
		boolToInt(feed.Enabled),
		feed.CreatedAt.Format(time.RFC3339),
		feed.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("insert feed: %w", err)
	}
	return nil
}

func (s *SQLStore) GetFeedByID(ctx context.Context, id string) (*Feed, error) {
	query := `
		SELECT id, user_id, name, secret, usernames, first_per_user, enabled, created_at, updated_at
		FROM feeds WHERE id = ?
	`
	return s.scanFeed(s.db.QueryRowContext(ctx, query, id))
}

func (s *SQLStore) GetFeedByIDAndSecret(ctx context.Context, id, secret string) (*Feed, error) {
	query := `
		SELECT id, user_id, name, secret, usernames, first_per_user, enabled, created_at, updated_at
		FROM feeds WHERE id = ? AND secret = ?
	`
	return s.scanFeed(s.db.QueryRowContext(ctx, query, id, secret))
}

func (s *SQLStore) UpdateFeed(ctx context.Context, feed *Feed) error {
	usernamesJSON, err := json.Marshal(feed.Usernames)
	if err != nil {
		return fmt.Errorf("marshal usernames: %w", err)
	}

	query := `
		UPDATE feeds 
		SET name = ?, secret = ?, usernames = ?, first_per_user = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := s.db.ExecContext(ctx, query,
		feed.Name,
		feed.Secret,
		string(usernamesJSON),
		feed.FirstPerUser,
		boolToInt(feed.Enabled),
		feed.UpdatedAt.Format(time.RFC3339),
		feed.ID,
	)
	if err != nil {
		return fmt.Errorf("update feed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLStore) DeleteFeed(ctx context.Context, id string) error {
	query := `DELETE FROM feeds WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete feed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLStore) ListFeedsByUserID(ctx context.Context, userID string) ([]Feed, error) {
	query := `
		SELECT id, user_id, name, secret, usernames, first_per_user, enabled, created_at, updated_at
		FROM feeds WHERE user_id = ? ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query feeds: %w", err)
	}
	defer rows.Close()

	var feeds []Feed
	for rows.Next() {
		feed, err := s.scanFeedFromRows(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, *feed)
	}
	return feeds, rows.Err()
}

func (s *SQLStore) CountFeedsByUserID(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM feeds WHERE user_id = ?`
	var count int
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count feeds: %w", err)
	}
	return count, nil
}

func (s *SQLStore) scanFeed(row *sql.Row) (*Feed, error) {
	var feed Feed
	var usernamesJSON string
	var enabled int
	var createdAt, updatedAt string

	err := row.Scan(
		&feed.ID,
		&feed.UserID,
		&feed.Name,
		&feed.Secret,
		&usernamesJSON,
		&feed.FirstPerUser,
		&enabled,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan feed: %w", err)
	}

	if err := json.Unmarshal([]byte(usernamesJSON), &feed.Usernames); err != nil {
		return nil, fmt.Errorf("unmarshal usernames: %w", err)
	}
	feed.Enabled = enabled == 1
	feed.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	feed.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &feed, nil
}

func (s *SQLStore) scanFeedFromRows(rows *sql.Rows) (*Feed, error) {
	var feed Feed
	var usernamesJSON string
	var enabled int
	var createdAt, updatedAt string

	err := rows.Scan(
		&feed.ID,
		&feed.UserID,
		&feed.Name,
		&feed.Secret,
		&usernamesJSON,
		&feed.FirstPerUser,
		&enabled,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan feed: %w", err)
	}

	if err := json.Unmarshal([]byte(usernamesJSON), &feed.Usernames); err != nil {
		return nil, fmt.Errorf("unmarshal usernames: %w", err)
	}
	feed.Enabled = enabled == 1
	feed.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	feed.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &feed, nil
}

// --- Feed cache operations ---

func (s *SQLStore) GetFeedCache(ctx context.Context, feedID string) (*FeedCache, error) {
	query := `
		SELECT feed_id, xml, etag, last_built_at, expires_at, last_error
		FROM feed_cache WHERE feed_id = ?
	`
	var cache FeedCache
	var lastBuiltAt, expiresAt sql.NullString

	err := s.db.QueryRowContext(ctx, query, feedID).Scan(
		&cache.FeedID,
		&cache.XML,
		&cache.ETag,
		&lastBuiltAt,
		&expiresAt,
		&cache.LastError,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan feed cache: %w", err)
	}

	if lastBuiltAt.Valid {
		cache.LastBuiltAt, _ = time.Parse(time.RFC3339, lastBuiltAt.String)
	}
	if expiresAt.Valid {
		cache.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt.String)
	}
	return &cache, nil
}

func (s *SQLStore) SetFeedCache(ctx context.Context, cache *FeedCache) error {
	query := `
		INSERT INTO feed_cache (feed_id, xml, etag, last_built_at, expires_at, last_error)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(feed_id) DO UPDATE SET
			xml = excluded.xml,
			etag = excluded.etag,
			last_built_at = excluded.last_built_at,
			expires_at = excluded.expires_at,
			last_error = excluded.last_error
	`
	_, err := s.db.ExecContext(ctx, query,
		cache.FeedID,
		cache.XML,
		cache.ETag,
		cache.LastBuiltAt.Format(time.RFC3339),
		cache.ExpiresAt.Format(time.RFC3339),
		cache.LastError,
	)
	if err != nil {
		return fmt.Errorf("upsert feed cache: %w", err)
	}
	return nil
}

func (s *SQLStore) InvalidateFeedCache(ctx context.Context, feedID string) error {
	query := `DELETE FROM feed_cache WHERE feed_id = ?`
	_, err := s.db.ExecContext(ctx, query, feedID)
	if err != nil {
		return fmt.Errorf("delete feed cache: %w", err)
	}
	return nil
}

// isUniqueConstraintError checks if the error is a unique constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed")
}

// boolToInt converts a boolean to SQLite integer (0 or 1).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
