APP_NAME=goapi
BIN_DIR=bin
BIN_PATH=$(BIN_DIR)/$(APP_NAME)

.PHONY: build run test test-unit test-integration clean docker-build

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

docker-build:
	docker build -t $(APP_NAME):local .

clean:
	rm -rf $(BIN_DIR)

docker-compose-up:
	docker compose up -d --build

docker-compose-down:
	docker compose down