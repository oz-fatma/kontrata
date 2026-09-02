.PHONY: run build test test-akis test-org lint tidy generate verify

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# Make her tarif satırını ayrı kabukta çalıştırır; yükleme ile go test
# aynı satırda kalmalı. Değişken atamasında # yorum olduğu için \# şart.
# Ortamda zaten tanımlı değişken .env ile ezilmez.
load_dotenv = if [ -f .env ]; then while IFS= read -r line || [ -n "$$line" ]; do case "$$line" in ''|'\#'*) continue ;; *=*) key=$${line%%=*}; val=$${line\#*=}; if [ -n "$$key" ] && ! printenv "$$key" >/dev/null; then export "$$key=$$val"; fi ;; esac; done < .env; fi

run:
	go run ./cmd/api

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/api ./cmd/api

test:
	@$(load_dotenv); go test ./...

test-akis:
	@$(load_dotenv); go test ./graph -count=1 -v -timeout 2m -run '^TestAuthAkis$$'

test-org:
	@$(load_dotenv); go test ./graph -count=1 -v -timeout 2m -run '^TestOrganizasyonAkis$$'

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
	@$(load_dotenv); go test ./...
