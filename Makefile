# Makefile for building hx-ollama static binaries

BINARY_NAME=hx-ollama
BUILD_DIR=dist

.PHONY: all build clean release

all: build

build:
	go build -ldflags="-s -w" -o bin/$(BINARY_NAME) main.go

release:
	mkdir -p $(BUILD_DIR)
	# macOS Apple Silicon (arm64)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 main.go
	# macOS Intel (amd64)
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 main.go
	# Linux (amd64 / Arch Linux)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go
	# Linux ARM64 (Raspberry Pi / Arch ARM)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 main.go

clean:
	rm -rf bin/$(BINARY_NAME) $(BUILD_DIR)
