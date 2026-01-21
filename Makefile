.PHONY: help run build test fmt tidy migrate-up migrate-down migrate-status migrate-create

.DEFAULT_GOAL := help

ifneq (,$(wildcard .env))
include .env
export
endif

DATABASE_URL ?= file:./data/leetrss.db?_journal=WAL&_timeout=5000

ifneq (,$(findstring libsql://,$(DATABASE_URL)))
MIGRATE_DRIVER := turso
else
MIGRATE_DRIVER := sqlite3
endif

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  help             Show this help message"
	@echo "  run              Start the development API server"
	@echo "  build            Build binary to bin/api"
	@echo "  test             Run all tests"
	@echo "  fmt              Format Go source files"
	@echo "  tidy             Clean up go.mod dependencies"
	@echo "  migrate-up       Apply pending migrations"
	@echo "  migrate-down     Rollback last migration"
	@echo "  migrate-status   Show migration status"
	@echo "  migrate-create   Create new migration (usage: make migrate-create NAME=add_users)"

run:
	@CGO_ENABLED=1 go run ./cmd/api

build:
	@CGO_ENABLED=1 go build -o bin/api ./cmd/api

test:
	@go test ./... -mod=readonly

fmt:
	@gofmt -w ./cmd ./internal

tidy:
	@go mod tidy

migrate-up:
	@goose -dir migrations $(MIGRATE_DRIVER) "$(DATABASE_URL)" up

migrate-down:
	@goose -dir migrations $(MIGRATE_DRIVER) "$(DATABASE_URL)" down

migrate-status:
	@goose -dir migrations $(MIGRATE_DRIVER) "$(DATABASE_URL)" status

migrate-create:
	@goose -dir migrations create $(NAME) sql
