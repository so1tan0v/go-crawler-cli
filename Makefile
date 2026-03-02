BIN_DIR := bin
BIN := $(BIN_DIR)/so1-crawler

.PHONY: test build run multios-build

test:
	go test ./...

run:
	@if [ -z "$(URL)" ]; then \
		echo "URL is required. Example: make run URL=https://example.com"; \
		go run ./cmd/so1-crawler --help; \
	else \
		go run ./cmd/so1-crawler "$(URL)"; \
	fi

lint: 
	golangci-lint run ./...

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) ./cmd/so1-crawler

multios-build:
	mkdir -p $(BIN_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/so1-crawler-linux-amd64 ./cmd/so1-crawler
	# MacOS
	GOOS=darwin GOARCH=amd64 go build -o $(BIN_DIR)/so1-crawler-darwin-amd64 ./cmd/so1-crawler
	# Windows
	GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/so1-crawler-windows-amd64.exe ./cmd/so1-crawler