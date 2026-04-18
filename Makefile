APP_NAME=goapi
BIN_DIR=bin
BIN_PATH=$(BIN_DIR)/$(APP_NAME)

.PHONY: build run test test-unit test-integration test-testcontainers coverage coverage-check clean docker-build

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_PATH) ./cmd/api

run:
	go run ./cmd/api

test: test-unit test-integration

test-unit:
	go test ./...

test-integration:
	go test -tags=integration ./...

# Requires Docker (pulls postgres image). Skipped when -short is set.
test-testcontainers:
	go test -tags=testcontainers ./internal/...

docker-build:
	docker build -t $(APP_NAME):local .

clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out

# Requires Docker for postgres testcontainers (same packages as CI).
# Omit cmd/api in package list due to Go 1.25+coverage toolchain quirk on some builds.
COVERAGE_PACKAGES := ./api ./internal/...
COVERAGE_TAGS := integration testcontainers

coverage:
	go test -tags='$(COVERAGE_TAGS)' -coverprofile=coverage.out -covermode=atomic $(COVERAGE_PACKAGES)
	@go tool cover -func=coverage.out | awk '/^total:/ { print "total coverage:", $$3 }'

coverage-check:
	go test -tags='$(COVERAGE_TAGS)' -coverprofile=coverage.out -covermode=atomic $(COVERAGE_PACKAGES)
	@COVERAGE=$$(go tool cover -func=coverage.out | awk '/^total:/ { gsub(/%/, "", $$3); print $$3 }'); \
	echo "total coverage: $${COVERAGE}% (minimum 85%)"; \
	awk -v c="$$COVERAGE" 'BEGIN { if (c+0 < 85) { print "coverage below 85%"; exit 1 } }'

docker-compose-up:
	docker compose up -d --build

docker-compose-down:
	docker compose down