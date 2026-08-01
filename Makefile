# Makefile for building hx-ollama static binaries (C and Go)

BINARY_NAME=hx-ollama
CC?=gcc
CFLAGS?=-O3 -Wall

.PHONY: all c-build go-build clean release

all: c-build

c-build:
	$(CC) $(CFLAGS) hx-ollama.c cJSON.c -o bin/$(BINARY_NAME)

go-build:
	go build -ldflags="-s -w" -o bin/$(BINARY_NAME) main.go

clean:
	rm -rf bin/$(BINARY_NAME) bin/hx-ollama-c dist

