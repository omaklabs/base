.PHONY: dev build generate lint test

dev:
	air

build:
	templ generate
	go build -o bin/app ./cmd/app

generate:
	templ generate
	sqlc generate

lint:
	golangci-lint run

test:
	go test ./...
