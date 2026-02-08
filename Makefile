BIN_DIR := bin/cmd

URL := https://hexlet.io/courses


.PHONY: test build

test:
	go test ./...

build:
	go build -o $(BIN_DIR)/main.go ./cmd/hexlet-go-crawler/main.go

run:
	./$(BIN_DIR)/main --url=${URL}