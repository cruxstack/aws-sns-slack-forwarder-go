.PHONY: help build-lambda clean test test-unit test-verbose test-coverage test-verify test-verify-verbose fmt lint tidy

# Default target
help:
	@echo "Available targets:"
	@echo "  build-lambda      - Build Lambda function binary"
	@echo "  package-lambda    - Build and package Lambda function as zip"
	@echo "  clean             - Remove build artifacts"
	@echo "  test              - Run all tests (unit + integration)"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-verbose      - Run tests with verbose output"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  test-verify       - Run integration tests"
	@echo "  test-verify-verbose - Run integration tests with verbose output"
	@echo "  fmt               - Format Go code"
	@echo "  lint              - Run linters"
	@echo "  tidy              - Tidy and verify dependencies"

# Build Lambda function
build-lambda:
	@echo "Building Lambda function..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -ldflags="-s -w" -o dist/bootstrap ./cmd/lambda
	@echo "Build complete: dist/bootstrap"

# Package Lambda function
package-lambda: build-lambda
	@echo "Packaging Lambda function..."
	@cd dist && zip -q function.zip bootstrap
	@echo "Package complete: dist/function.zip"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf dist/
	@echo "Clean complete"

# Run all tests
test:
	@echo "Running unit tests..."
	go test -race -count=1 ./...
	@echo "Running integration tests..."
	go run ./cmd/verify

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	go test -race -count=1 ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	go test -v -race -count=1 ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
		echo "Install: https://golangci-lint.run/usage/install/"; \
	fi

# Run integration tests
test-verify:
	@echo "Running integration tests..."
	go run ./cmd/verify

# Run integration tests with verbose output
test-verify-verbose:
	@echo "Running integration tests (verbose)..."
	go run ./cmd/verify -verbose

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy
	go mod verify
