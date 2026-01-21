FROM golang:1.25-alpine AS build

WORKDIR /app

# Install build dependencies for CGO (required by go-libsql)
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /app/bin/api ./cmd/api

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=build /app/bin/api ./api

EXPOSE 8080

CMD ["./api"]
