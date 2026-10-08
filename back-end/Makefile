# Packages measured for coverage: everything except the main() wiring.
COVER_PKGS := ./internal/...,./cmd/webapp/routes/...

.PHONY: run dev test test-unit test-integration cover lint

run:
	go run ./cmd/webapp

# Hot reload: rebuilds and restarts on every .go change.
# Install once with: go install github.com/air-verse/air@latest
dev:
	$(shell go env GOPATH)/bin/air

test: test-unit test-integration

test-unit:
	go test -race ./test/unit/...

test-integration:
	go test -race ./test/integration/...

# Runs every test and enforces 100% coverage of COVER_PKGS.
cover:
	go test -race -coverpkg=$(COVER_PKGS) -coverprofile=coverage.out ./test/...
	./scripts/check-coverage.sh

lint:
	go vet ./...
	test -z "$$(gofmt -l .)"
