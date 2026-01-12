.PHONY: run test fmt tidy migrate-up migrate-down migrate-status migrate-create

# Default database URL for local development
DATABASE_URL ?= file:./data/leetrss.db?_journal=WAL&_timeout=5000

run:
	@go run ./cmd/api

test:
	@go test ./... -mod=readonly

fmt:
	@gofmt -w ./cmd ./internal

tidy:
	@go mod tidy

# Database migrations (requires goose: go install github.com/pressly/goose/v3/cmd/goose@latest)
migrate-up:
	@goose -dir migrations sqlite3 "$(DATABASE_URL)" up

migrate-down:
	@goose -dir migrations sqlite3 "$(DATABASE_URL)" down

migrate-status:
	@goose -dir migrations sqlite3 "$(DATABASE_URL)" status

migrate-create:
	@goose -dir migrations create $(NAME) sql
