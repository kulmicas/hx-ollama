# Makefile for hx-ollama (Supports Go and C static builds)

BINARY_NAME=hx-ollama
INSTALL_DIR=$(HOME)/.local/bin
GO_EXISTS := $(shell command -v go 2> /dev/null)

.PHONY: all build install clean

all: build

build:
ifdef GO_EXISTS
	@echo "📦 Building with Go..."
	@mkdir -p bin
	CGO_ENABLED=0 go build -o bin/$(BINARY_NAME) main.go
	@echo "✅ Built bin/$(BINARY_NAME) using Go successfully."
else
	@echo "📦 Go not found. Building with C compiler (gcc/clang)..."
	@mkdir -p bin
	$(CC) -O3 -Wall hx-ollama.c cJSON.c -o bin/$(BINARY_NAME)
	@echo "✅ Built bin/$(BINARY_NAME) using C successfully."
endif

install: build
	@mkdir -p $(INSTALL_DIR)
	cp bin/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "⚙️ Initializing configuration..."
	$(INSTALL_DIR)/$(BINARY_NAME) setup
	@echo "✅ Installed hx-ollama binary to $(INSTALL_DIR)/$(BINARY_NAME)"

clean:
	rm -rf bin/
