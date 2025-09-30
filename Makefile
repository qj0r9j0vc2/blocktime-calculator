BINARY_NAME=blocktime-calculator
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

.PHONY: all build clean test coverage deps run install

all: clean deps build

build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) cmd/main.go

clean:
	@echo "Cleaning..."
	@if [ -f $(BINARY_NAME) ]; then rm $(BINARY_NAME); fi
	@go clean

deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

test:
	@echo "Running tests..."
	@go test -v ./...

coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

run:
	@go run cmd/main.go $(ARGS)

install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) ./cmd

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

lint:
	@echo "Running linter..."
	@golangci-lint run ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

help:
	@echo "Available targets:"
	@echo "  make build     - Build the binary"
	@echo "  make clean     - Clean build artifacts"
	@echo "  make deps      - Download dependencies"
	@echo "  make test      - Run tests"
	@echo "  make coverage  - Generate test coverage report"
	@echo "  make run       - Run the application (use ARGS='...' to pass arguments)"
	@echo "  make install   - Install the binary"
	@echo "  make fmt       - Format code"
	@echo "  make lint      - Run linter"
	@echo "  make vet       - Run go vet"
	@echo "  make help      - Show this help message"