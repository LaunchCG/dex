.PHONY: build test test-cover fmt vet lint clean install install-user help

BINARY_NAME := dex
GOPATH := $(shell go env GOPATH)
BIN_DIR := bin

# Default target
all: build

## build: Build the CLI binary to bin/dex
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/dex

## test: Run all tests
test:
	go test ./...

## test-cover: Run tests with coverage report
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

## fmt: Format code with go fmt
fmt:
	go fmt ./...

## vet: Run go vet
vet:
	go vet ./...

## lint: Run fmt + vet
lint: fmt vet

## clean: Remove built binary and coverage files
clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html

## install: Install binary to GOPATH/bin
install: build
	cp $(BIN_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

## install-user: Install binary to ~/.bin
install-user: build
	@mkdir -p $(HOME)/.bin
	cp $(BIN_DIR)/$(BINARY_NAME) $(HOME)/.bin/
	@echo "Installed to $(HOME)/.bin/$(BINARY_NAME)"
	@echo "Make sure $(HOME)/.bin is in your PATH"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'
