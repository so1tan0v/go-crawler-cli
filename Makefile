BIN_DIR := bin
BIN := $(BIN_DIR)/hexlet-go-crawler

.PHONY: test build run

test:
	go test ./...

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) ./cmd/hexlet-go-crawler

run:
	@if [ -z "$(URL)" ]; then \
		echo "URL is required. Example: make run URL=https://example.com"; \

		go run ./cmd/hexlet-go-crawler --help; \
	else \
		go run ./cmd/hexlet-go-crawler "$(URL)"; \
	fi

lint: 
	golangci-lint run ./...