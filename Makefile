.PHONY: run test fmt tidy

run:
	@go run ./cmd/api

test:
	@go test ./... -mod=readonly

fmt:
	@gofmt -w ./cmd ./internal

tidy:
	@go mod tidy
