.PHONY: build test lint run sqlc

build:
	go build -o cmd/oversite/oversite ./cmd/oversite

run:
	go run ./cmd/oversite serve

test:
	go test -race ./...

test-integration:
	go test -race -tags integration ./...

lint:
	go tool golangci-lint run ./...

sqlc:
	go tool sqlc generate
