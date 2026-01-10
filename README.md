# leet-rss (LeetCode RSS)

A small Go HTTP service that generates an RSS 2.0 feed from one or more users' LeetCode Solution Articles (Discuss) using LeetCode’s GraphQL API.

## What is there

- RSS feed endpoint: `GET /leetcode.xml`
- Health endpoint: `GET /health`
- In-memory TTL cache for the generated RSS
- Optional support for authenticated requests via `LEETCODE_COOKIE` and `LEETCODE_CSRF`

## Project Layout

The Go module lives under `leetcode-rss/`:

- `leetcode-rss/cmd/api/`: server entrypoint and routes
- `leetcode-rss/internal/api/`: handlers, feed service, cache
- `leetcode-rss/internal/leetcode/`: GraphQL client, query, models
- `leetcode-rss/internal/rss/`: RSS structs and XML rendering
- `leetcode-rss/.env.example`: example local configuration

## Requirements

- Go version specified in `leetcode-rss/go.mod`
- Make (optional, for the provided `Makefile`)

## Configuration

Create a local env file at `leetcode-rss/.env` (see `leetcode-rss/.env.example`):

```dotenv
LEETCODE_USERNAMES=user_one,user_two,user_three
PORT=8080
CACHE_TTL=2m
HANDLER_TIMEOUT=10s
LEETCODE_MAX_ARTICLES=15
LEETCODE_GRAPHQL_ENDPOINT=https://leetcode.com/graphql/
LEETCODE_COOKIE=
LEETCODE_CSRF=
```

Environment variables:

- `LEETCODE_USERNAMES` (required): comma-separated list of LeetCode usernames to generate the feed for
- `PORT` (default `8080`): server listen port
- `HANDLER_TIMEOUT` (default `10s`): per-request handler timeout (Go duration format)
- `CACHE_TTL` (default `2m`): in-memory cache TTL (Go duration format, e.g. `30s`, `5m`)
- `LEETCODE_MAX_ARTICLES` (default `15`): max solution articles fetched per user (clamped to `1..50`)
- `LEETCODE_GRAPHQL_ENDPOINT` (default `https://leetcode.com/graphql/`): GraphQL endpoint
- `LEETCODE_COOKIE` (optional): cookie header value for authenticated requests
- `LEETCODE_CSRF` (optional): CSRF token for authenticated requests

## Run Locally

From the module directory:

```bash
cd leetcode-rss
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

- `make test`: `go test ./... -mod=readonly`
- `make fmt`: format with `gofmt`
- `make tidy`: run `go mod tidy`

### Testing Notes

- Uses Go’s standard `testing` package.
- Add tests as `*_test.go` files next to the code under `leetcode-rss/internal/...`.

## Troubleshooting

- `missing env LEETCODE_USERNAMES`: set `LEETCODE_USERNAMES` in the env or `leetcode-rss/.env`.
- `leetcode http 4xx/5xx` or `graphql error`: LeetCode may be rate-limiting, blocking, or returning an error. Increase `CACHE_TTL` and consider setting `LEETCODE_COOKIE`/`LEETCODE_CSRF` if your feed requires authentication.
- RSS link looks wrong: solution links rely on `questionSlug` returned by the API; if LeetCode changes response fields, the link format may need updating.


## Disclaimer

**⚠️This project is not affiliated with LeetCode in any way**. LeetCode may change APIs or page structures at any time, which can affect feed generation.

