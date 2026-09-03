.PHONY: run build build-darwin build-windows test test-akis test-org lint tidy generate verify

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# Make her tarif satırını ayrı kabukta çalıştırır; yükleme ile go test
# aynı satırda kalmalı. Değişken atamasında # yorum olduğu için \# şart.
# Ortamda zaten tanımlı değişken .env ile ezilmez.
load_dotenv = if [ -f .env ]; then while IFS= read -r line || [ -n "$$line" ]; do case "$$line" in ''|'\#'*) continue ;; *=*) key=$${line%%=*}; val=$${line\#*=}; if [ -n "$$key" ] && ! printenv "$$key" >/dev/null; then export "$$key=$$val"; fi ;; esac; done < .env; fi

run:
	go run ./cmd/api

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/api ./cmd/api

# Electron paketlemesi için platform ikilileri (GOOS/GOARCH).
# CGO kapalı: macOS'tan Windows ve çapraz Darwin derlemesi saf Go kalır.
build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o bin/api-darwin-arm64 ./cmd/api
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o bin/api-darwin-amd64 ./cmd/api

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o bin/api-windows-amd64.exe ./cmd/api

test:
	@$(load_dotenv); go test ./...

test-akis:
	@$(load_dotenv); go test ./graph -count=1 -v -timeout 2m -run '^TestAuthFlow$$'

test-org:
	@$(load_dotenv); go test ./graph -count=1 -v -timeout 2m -run '^TestOrganizationFlow$$'

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
