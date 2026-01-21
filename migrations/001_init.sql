-- +goose Up
-- Initial schema for multi-tenant LeetCode RSS service

-- Users table: stores authenticated users
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    auth_provider TEXT,
    provider_subject TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Unique constraint on provider credentials (for OAuth lookup)
CREATE UNIQUE INDEX idx_users_provider ON users(auth_provider, provider_subject)
    WHERE auth_provider IS NOT NULL;

-- Feeds table: stores user-created RSS feed configurations
CREATE TABLE feeds (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    secret TEXT NOT NULL,
    usernames TEXT NOT NULL,  -- JSON array of LeetCode usernames
    first_per_user INTEGER NOT NULL DEFAULT 15 CHECK (first_per_user >= 1 AND first_per_user <= 50),
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Index for listing feeds by user
CREATE INDEX idx_feeds_user_id ON feeds(user_id);

-- Unique index for public feed URL lookup (id + secret)
CREATE UNIQUE INDEX idx_feeds_id_secret ON feeds(id, secret);

-- Feed cache table: stores cached RSS XML for each feed
CREATE TABLE feed_cache (
    feed_id TEXT PRIMARY KEY REFERENCES feeds(id) ON DELETE CASCADE,
    xml BLOB,
    etag TEXT,
    last_built_at TEXT,
    expires_at TEXT,
    last_error TEXT
);

-- +goose Down
DROP TABLE IF EXISTS feed_cache;
DROP TABLE IF EXISTS feeds;
DROP TABLE IF EXISTS users;
