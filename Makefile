PROTO_DIR   := proto
API_DIR     := api/stats
BIN_DIR     := bin

.PHONY: all build run test lint generate clean

all: build

build:
	go build -o $(BIN_DIR)/sysmon-daemon ./cmd/daemon
	go build -o $(BIN_DIR)/sysmon-client ./cmd/client

run:
	go run ./cmd/daemon

test:
	go test -race -count 100 ./...

test-integration:
	go test -race -tags integration ./tests/integration/...

lint:
	golangci-lint run ./...

generate:
	protoc \
		--go_out=$(API_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(API_DIR) \
		--go-grpc_opt=paths=source_relative \
		--proto_path=$(PROTO_DIR) \
		stats.proto

clean:
	rm -rf $(BIN_DIR)
