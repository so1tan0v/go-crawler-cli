BIN_DIR := bin
BIN := $(BIN_DIR)/so1-crawler

.PHONY: test build run

test:
	go test ./...

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) ./cmd/so1-crawler

run:
	@if [ -z "$(URL)" ]; then \
		echo "URL is required. Example: make run URL=https://example.com"; \
		go run ./cmd/so1-crawler --help; \
	else \
		go run ./cmd/so1-crawler "$(URL)"; \
	fi

lint: 
	golangci-lint run ./...