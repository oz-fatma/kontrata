.PHONY: run build test test-akis lint tidy generate verify

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

run:
	go run ./cmd/api

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/api ./cmd/api

test:
	go test ./...

test-akis:
	go test ./graph -count=1 -v -timeout 2m -run '^TestAuthAkis$$'

lint:
	go vet ./...

tidy:
	go mod tidy

generate:
	go tool github.com/99designs/gqlgen generate

verify:
	go mod tidy
	git diff --exit-code go.mod go.sum
	go vet ./...
	go test ./...
