package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

type Store interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByProvider(ctx context.Context, provider, subject string) (*User, error)

	CreateFeed(ctx context.Context, feed *Feed) error
	GetFeedByID(ctx context.Context, id string) (*Feed, error)
	GetFeedByIDAndSecret(ctx context.Context, id, secret string) (*Feed, error)
	UpdateFeed(ctx context.Context, feed *Feed) error
	DeleteFeed(ctx context.Context, id string) error
	ListFeedsByUserID(ctx context.Context, userID string) ([]Feed, error)
	CountFeedsByUserID(ctx context.Context, userID string) (int, error)

	GetFeedCache(ctx context.Context, feedID string) (*FeedCache, error)
	SetFeedCache(ctx context.Context, cache *FeedCache) error
	InvalidateFeedCache(ctx context.Context, feedID string) error

	Close() error
}

// supported DSN formats:
//
//	Local sqlite: "file:./data/leetrss.db" or ":memory:"
//	TursoDB: "libsql://[db-name]-[org].turso.io?authToken=..."
//
// NOTE: all formats are handled by the libsql driver which supports both local and remote.
func NewStore(dsn string) (Store, error) {
	switch {
	case strings.HasPrefix(dsn, "file:"), dsn == ":memory:", strings.HasPrefix(dsn, ":memory:"), strings.HasPrefix(dsn, "libsql://"):
		return NewSQLStore(dsn)
	default:
		return nil, fmt.Errorf("unsupported database DSN: %s (expected file:, :memory:, or libsql://)", dsn)
	}
}
