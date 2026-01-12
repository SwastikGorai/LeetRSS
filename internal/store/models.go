package store

import "time"

type User struct {
	ID              string
	Email           string
	AuthProvider    *string // "github", "google"..etc.. (nil for magic link)
	ProviderSubject *string // provider's user ID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Feed struct {
	ID           string
	UserID       string
	Name         string
	Secret       string
	Usernames    []string
	FirstPerUser int
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type FeedCache struct {
	FeedID      string
	XML         []byte
	ETag        string
	LastBuiltAt time.Time
	ExpiresAt   time.Time
	LastError   *string
}
