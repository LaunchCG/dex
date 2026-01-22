.PHONY: build test test-cover fmt vet lint clean install help

BINARY_NAME := dex
GOPATH := $(shell go env GOPATH)

# Default target
all: build

## build: Build the CLI binary to ./dex
build:
	go build -o $(BINARY_NAME) ./cmd/dex

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
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## install: Install binary to GOPATH/bin
install: build
	cp $(BINARY_NAME) $(GOPATH)/bin/

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'
