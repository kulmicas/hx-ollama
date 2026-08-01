# Makefile for hx-ollama (Pure C Static Binary)

BINARY_NAME = hx-ollama
CC ?= gcc
CFLAGS ?= -O3 -Wall
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: all build install clean

all: build

build:
	@mkdir -p bin
	$(CC) $(CFLAGS) hx-ollama.c cJSON.c -o bin/$(BINARY_NAME)

install: build
	@mkdir -p $(INSTALL_DIR)
	cp bin/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "⚙️ Initializing configuration..."
	$(INSTALL_DIR)/$(BINARY_NAME) setup
	@echo "✅ Installed hx-ollama binary to $(INSTALL_DIR)/$(BINARY_NAME)"

clean:
	rm -rf bin/
