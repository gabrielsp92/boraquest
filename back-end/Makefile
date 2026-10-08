# Packages measured for coverage: everything except the main() wiring.
COVER_PKGS := ./internal/...,./cmd/webapp/routes/...

.PHONY: run dev db-up db-down seed-user seed-user-dry-run test test-unit test-integration cover lint

run:
	go run ./cmd/webapp

# Dev-only default so `make run`/`make dev` work locally; always set a real secret elsewhere.
export JWT_SECRET ?= boraquest-dev-secret

# Local PostgreSQL (docker-compose.yml). The API reads DATABASE_URL and JWT_SECRET.
db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

# Adds one user from SEED_USER_NAME, SEED_USER_EMAIL and SEED_USER_PASSWORD (plus optional
# DATABASE_URL), read from $(ENV_FILE) when it exists, else from the environment. See .env.example.
# An existing email is left untouched. The dry run validates and checks the DB but writes nothing.
ENV_FILE ?= ./.env
load_env = set -a; [ ! -f $(ENV_FILE) ] || . $(ENV_FILE); set +a

seed-user:
	@$(load_env); go run ./cmd/seeduser

seed-user-dry-run:
	@$(load_env); go run ./cmd/seeduser -dry-run

# Hot reload: rebuilds and restarts on every .go change.
# Install once with: go install github.com/air-verse/air@latest
dev:
	$(shell go env GOPATH)/bin/air

test: test-unit test-integration

test-unit:
	go test -race ./test/unit/...

test-integration:
	go test -race ./test/integration/...

# Integration tests start their own PostgreSQL container (testcontainers), so Docker must be running.
# Runs every test and enforces 100% coverage of COVER_PKGS.
cover:
	go test -race -coverpkg=$(COVER_PKGS) -coverprofile=coverage.out ./test/...
	./scripts/check-coverage.sh

lint:
	go vet ./...
	test -z "$$(gofmt -l .)"
