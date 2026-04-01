.PHONY: build test lint run sqlc

build:
	go build -o cmd/oversite/oversite ./cmd/oversite

run:
	go run ./cmd/oversite serve

test:
	go test ./...

test-integration:
	go test -tags integration ./...

lint:
	golangci-lint run ./...

sqlc:
	sqlc generate
