.PHONY: test test-verbose test-short test-coverage test-race test-unit test-integration test-api help clean

# Default target
help:
	@echo "MulaMail 2 Server - Test Targets"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  test              - Run all tests"
	@echo "  test-verbose      - Run all tests with verbose output"
	@echo "  test-short        - Run tests in short mode (skip integration tests)"
	@echo "  test-coverage     - Run tests and generate coverage report"
	@echo "  test-race         - Run tests with race detector"
	@echo "  test-unit         - Run only unit tests (vault, config, blockchain)"
	@echo "  test-integration  - Run only integration tests (db)"
	@echo "  test-api          - Run only API handler tests"
	@echo "  coverage-html     - Generate and open HTML coverage report"
	@echo "  clean             - Remove test artifacts and coverage files"
	@echo "  help              - Show this help message"

# Run all tests
test:
	@echo "Running all tests..."
	go test ./...

# Run all tests with verbose output
test-verbose:
	@echo "Running all tests (verbose)..."
	go test -v ./...

# Run tests in short mode (skip integration tests)
test-short:
	@echo "Running tests in short mode (skipping integration tests)..."
	go test -short ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...

# Run tests with coverage and generate detailed report
coverage-report:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "Total coverage summary:"
	@go tool cover -func=coverage.out | grep total

# Generate and open HTML coverage report
coverage-html: coverage-report
	@echo "Opening HTML coverage report..."
	go tool cover -html=coverage.out

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race ./...

# Run only unit tests (no external dependencies)
test-unit:
	@echo "Running unit tests..."
	go test ./vault ./config ./blockchain

# Run only integration tests (requires MongoDB)
test-integration:
	@echo "Running integration tests..."
	@echo "Note: Requires MongoDB at mongodb://localhost:27017"
	go test -v ./db

# Run only API handler tests
test-api:
	@echo "Running API handler tests..."
	go test -v ./api

# Run tests for a specific package
test-vault:
	go test -v ./vault

test-config:
	go test -v ./config

test-db:
	go test -v ./db

test-blockchain:
	go test -v ./blockchain

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

bench-vault:
	go test -bench=. -benchmem ./vault

# Clean up test artifacts
clean:
	@echo "Cleaning test artifacts..."
	rm -f coverage.out
	rm -f *.test
	@echo "Done."

# CI/CD target - runs all tests with coverage and race detection
ci:
	@echo "Running CI test suite..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@echo "Starting MongoDB (Docker)..."
	docker run -d -p 27017:27017 --name mulamail-test-mongo mongo:latest
	@echo "MongoDB is running on port 27017"
	@echo ""
	@echo "Run 'make test' to verify setup"

# Teardown development environment
teardown:
	@echo "Tearing down development environment..."
	docker stop mulamail-test-mongo || true
	docker rm mulamail-test-mongo || true
	@echo "Done."

# Quick test (just unit tests, no verbose output)
quick:
	@go test ./vault ./config ./blockchain ./api

# Watch mode - rerun tests on file changes (requires entr or similar)
watch:
	@echo "Watching for changes... (requires 'entr' command)"
	@find . -name '*.go' | entr -c make quick
