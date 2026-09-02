.PHONY: run build test lint tidy generate

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

run:
	go run ./cmd/api

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/api ./cmd/api

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

generate:
	go tool github.com/99designs/gqlgen generate
