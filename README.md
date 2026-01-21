# leet-rss (LeetCode RSS)

A small Go HTTP service that generates an RSS 2.0 feed from one or more users' LeetCode Solution Articles (Discuss) using LeetCode's GraphQL API.

## What is there

- RSS feed endpoint: `GET /leetcode.xml`
- Health endpoint: `GET /health`
- In-memory TTL cache for the generated RSS
- Optional support for authenticated requests via `LEETCODE_COOKIE` and `LEETCODE_CSRF`
- Public per-feed RSS endpoint: `GET /f/:feedID/:secret.xml`
- Authenticated feed management API (requires Clerk): `GET /me`, `GET /feeds`, `POST /feeds`, `PATCH /feeds/:id`, `POST /feeds/:id/rotate`, `DELETE /feeds/:id`

## Project Layout

The Go module lives under `leetcode-rss/`:

- `leetcode-rss/cmd/api/`: server entrypoint and routes
- `leetcode-rss/internal/api/`: handlers, feed service, cache
- `leetcode-rss/internal/leetcode/`: GraphQL client, query, models
- `leetcode-rss/internal/rss/`: RSS structs and XML rendering
- `leetcode-rss/internal/store/`: database repository layer (SQLite/TursoDB)
- `leetcode-rss/migrations/`: database schema migrations (goose)
- `leetcode-rss/data/`: local SQLite database files
- `leetcode-rss/.env.example`: example local configuration

## Requirements

- Go version specified in `leetcode-rss/go.mod`
- Make (optional, for the provided `Makefile`)
- [goose](https://github.com/pressly/goose) for database migrations (optional for development)

## Quick Start

### 1. Install dependencies

```bash
# Install goose for database migrations
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### 2. Configure environment

```bash
cd leetcode-rss
cp .env.example .env
# edit .env with your settings
```

### 3. Run database migrations

```bash
make migrate-up
```

### 4. Start the server

```bash
make run
```

Then visit:
- `http://localhost:8080/` (basic info)
- `http://localhost:8080/health`
- `http://localhost:8080/leetcode.xml` (RSS)

### Quick Test (curl)

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/leetcode.xml
```

## Configuration

Create a local env file at `leetcode-rss/.env` (see `leetcode-rss/.env.example`):

```dotenv
# Server configuration
PORT=8080
HANDLER_TIMEOUT=10s

# LeetCode API settings
LEETCODE_USERNAMES=user_one,user_two,user_three
LEETCODE_USERNAME=user_one
LEETCODE_GRAPHQL_ENDPOINT=https://leetcode.com/graphql/
LEETCODE_MAX_ARTICLES=15
LEETCODE_COOKIE=
LEETCODE_CSRF=

# Cache settings
CACHE_TTL=2m

# Database configuration (SQLite for local dev)
DATABASE_URL=file:./data/leetrss.db?_journal=WAL&_timeout=5000

# Public URL for generating feed URLs
PUBLIC_BASE_URL=http://localhost:8080

# Per-feed RSS cache TTL
RSS_CACHE_TTL=5m

# Clerk authentication
CLERK_SECRET_KEY=

# Limits
MAX_FEEDS_PER_USER=3
MAX_USERNAMES_PER_FEED=3
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LEETCODE_USERNAMES` | (required) | Comma-separated list of LeetCode usernames |
| `PORT` | `8080` | Server listen port |
| `HANDLER_TIMEOUT` | `10s` | Per-request handler timeout (Go duration) |
| `CACHE_TTL` | `2m` | In-memory cache TTL (Go duration) |
| `LEETCODE_MAX_ARTICLES` | `15` | Max articles per user (clamped 1-50) |
| `LEETCODE_GRAPHQL_ENDPOINT` | `https://leetcode.com/graphql/` | GraphQL endpoint |
| `LEETCODE_COOKIE` | (optional) | Cookie header for authenticated requests |
| `LEETCODE_CSRF` | (optional) | CSRF token for authenticated requests |
| `DATABASE_URL` | `file:./data/leetrss.db?...` | SQLite or TursoDB connection string |
| `PUBLIC_BASE_URL` | `http://localhost:8080` | Base URL for feed URLs in API responses |
| `RSS_CACHE_TTL` | `5m` | Per-feed cache TTL for multi-tenant feeds |
| `CLERK_SECRET_KEY` | (optional) | Enables Clerk auth for protected feed endpoints |
| `MAX_FEEDS_PER_USER` | `3` | Max number of feeds per user (clamped 1-100) |
| `MAX_USERNAMES_PER_FEED` | `3` | Max usernames per feed (clamped 1-20) |

## Authentication (Clerk)

If `CLERK_SECRET_KEY` is set, the service enables protected API routes and verifies Clerk JWTs:

- `GET /me`: returns the current user record
- `GET /feeds`: list feeds for the user
- `POST /feeds`: create a new feed
- `PATCH /feeds/:id`: update feed settings
- `POST /feeds/:id/rotate`: rotate the feed secret
- `DELETE /feeds/:id`: delete a feed

When the secret key is missing, these routes are not registered.

## Database

The service uses SQLite for local development and can use [TursoDB](https://turso.tech) for production.

### Local Development (SQLite)

```bash
# Run migrations
make migrate-up

# Check migration status
make migrate-status

# Rollback last migration
make migrate-down

# Create a new migration
make migrate-create NAME=add_new_table
```

### Production (TursoDB)

```bash
# Install Turso CLI
curl -sSfL https://get.tur.so/install.sh | bash

# create za database
turso db create leetrss-prod

# get connection URL
turso db show --url leetrss-prod

# create auth token
turso db tokens create leetrss-prod

# Set DATABASE_URL in prod environment
# DATABASE_URL=libsql://name-prod-myorg.turso.io?authToken=eyJ...

# Run migrations with goose turso driver
goose -dir migrations turso "$DATABASE_URL" up
```

## How It Works

1. `cmd/api/main.go` loads config from environment (and `.env` if present).
2. The service calls LeetCode GraphQL to fetch the most recent solution articles for each configured user (currently `15` per user).
3. Articles from all users are merged and sorted by creation date(most recent first).
4. Each article is mapped to an RSS `<item>` with:
   - `title`: article title
   - `link`: solution permalink, e.g. `https://leetcode.com/problems/{questionSlug}/solutions/{topicId}/{slug}/`
   - `guid`: stable identifier based on topic and uuid
   - `pubDate`: article creation time
5. The rendered XML is cached for `CACHE_TTL`.

## Development

Run commands from `leetcode-rss/`:

- `make run`: start the server
- `make test`: run tests (`go test ./... -mod=readonly`)
- `make fmt`: format with `gofmt`
- `make tidy`: run `go mod tidy`
- `make migrate-up`: run database migrations
- `make migrate-down`: rollback last migration
- `make migrate-status`: show migration status

### Testing Notes

- Uses Go’s standard `testing` package.
- Add tests as `*_test.go` files next to the code under `leetcode-rss/internal/...`.

## Troubleshooting

- `missing env LEETCODE_USERNAMES`: set `LEETCODE_USERNAMES` in the env or `leetcode-rss/.env`.
- `leetcode http 4xx/5xx` or `graphql error`: LeetCode may be rate-limiting, blocking, or returning an error. Increase `CACHE_TTL` and consider setting `LEETCODE_COOKIE`/`LEETCODE_CSRF` if your feed requires authentication.
- RSS link looks wrong: solution links rely on `questionSlug` returned by the API; if LeetCode changes response fields, the link format may need updating.


## Disclaimer

**⚠️This project is not affiliated with LeetCode in any way**. LeetCode may change APIs or page structures at any time, which can affect feed generation.

