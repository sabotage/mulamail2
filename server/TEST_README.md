# MulaMail 2 Server - Testing Guide

## Overview

This directory contains comprehensive tests for all server components:

- **Unit tests**: Pure function tests with no external dependencies
- **Integration tests**: Tests that require external services (MongoDB, Solana RPC)
- **API tests**: HTTP handler tests using `httptest`

## Test Structure

```
server/
├── vault/
│   ├── encrypt.go
│   └── encrypt_test.go       # Unit tests for AES-GCM encryption
├── config/
│   ├── config.go
│   └── config_test.go        # Unit tests for config loading
├── db/
│   ├── mongo.go
│   ├── errors.go
│   └── mongo_test.go         # Integration tests for MongoDB operations
├── blockchain/
│   ├── identity.go
│   └── identity_test.go      # Unit tests for Solana identity transactions
├── api/
│   ├── router.go
│   ├── identity.go
│   ├── mail.go
│   ├── router_test.go        # HTTP handler tests (router & helpers)
│   ├── identity_test.go      # HTTP handler tests (identity endpoints)
│   └── mail_test.go          # HTTP handler tests (mail endpoints)
└── testutil/
    └── testutil.go           # Shared test utilities and helpers
```

## Running Tests

### Run All Tests

```bash
cd server
go test ./...
```

### Run Tests with Verbose Output

```bash
go test -v ./...
```

### Run Tests with Coverage

```bash
go test -cover ./...
```

### Generate Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Specific Package Tests

```bash
# Vault tests (encryption)
go test ./vault

# Config tests
go test ./config

# Database tests
go test ./db

# Blockchain tests
go test ./blockchain

# API tests
go test ./api
```

### Run Specific Test

```bash
go test ./vault -run TestEncryptAESGCM_Success
```

### Run in Short Mode (Skip Integration Tests)

```bash
go test -short ./...
```

## Test Categories

### 1. Unit Tests (No External Dependencies)

**vault/encrypt_test.go**
- Tests AES-256-GCM encryption/decryption
- Round-trip tests
- Error handling (invalid keys, tampered ciphertext)
- Benchmarks

**config/config_test.go**
- Environment variable loading
- Default values
- Partial configuration

**blockchain/identity_test.go**
- Memo instruction creation
- Transaction structure validation
- JSON payload formatting

### 2. Integration Tests (Require External Services)

**db/mongo_test.go**
- Requires: MongoDB instance
- Tests all CRUD operations
- Identity and MailAccount models
- Multi-document queries

#### Running MongoDB Integration Tests

**Option 1: Use local MongoDB**
```bash
# Start MongoDB (Docker)
docker run -d -p 27017:27017 --name mulamail-test-mongo mongo:latest

# Run tests
go test ./db

# Stop MongoDB
docker stop mulamail-test-mongo
docker rm mulamail-test-mongo
```

**Option 2: Use custom MongoDB URI**
```bash
export MONGO_TEST_URI="mongodb://your-test-instance:27017"
go test ./db
```

**Note**: Database tests automatically:
- Create a unique test database for each run
- Clean up (drop database) after tests complete
- Skip if MongoDB is not available

### 3. API/Handler Tests

**api/router_test.go**, **api/identity_test.go**, **api/mail_test.go**
- Use `httptest` for HTTP request/response testing
- Mock database and external dependencies
- Test all endpoints with various scenarios
- Error cases and edge cases

## Environment Variables for Testing

| Variable | Default | Description |
|----------|---------|-------------|
| `MONGO_TEST_URI` | `mongodb://localhost:27017` | MongoDB URI for integration tests |
| `SOLANA_TEST_RPC` | `https://api.devnet.solana.com` | Solana RPC endpoint for blockchain tests |

## Test Conventions

### Naming

- Test files: `*_test.go`
- Test functions: `TestXxx(t *testing.T)`
- Benchmark functions: `BenchmarkXxx(b *testing.B)`
- Helper functions: Mark with `t.Helper()`

### Structure

```go
func TestFeature_Scenario(t *testing.T) {
    // Setup

    // Execute

    // Assert

    // Cleanup (if needed, prefer t.Cleanup())
}
```

### Table-Driven Tests

```go
func TestMultipleScenarios(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
    }{
        {"scenario 1", "input1", "output1"},
        {"scenario 2", "input2", "output2"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      mongodb:
        image: mongo:latest
        ports:
          - 27017:27017

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Run tests
        run: |
          cd server
          go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./server/coverage.out
```

## Test Coverage Goals

| Package | Current Coverage | Target |
|---------|------------------|--------|
| vault | ~95% | 90%+ |
| config | ~95% | 90%+ |
| db | ~85% | 80%+ |
| blockchain | ~75% | 70%+ |
| api | ~80% | 80%+ |

## Common Issues

### MongoDB Connection Fails

**Problem**: Tests skip with "MongoDB not available"

**Solutions**:
1. Start MongoDB: `docker run -d -p 27017:27017 mongo:latest`
2. Check MongoDB is running: `mongosh --eval "db.version()"`
3. Set custom URI: `export MONGO_TEST_URI="mongodb://localhost:27017"`

### Solana RPC Timeouts

**Problem**: Blockchain tests timeout or fail

**Solutions**:
1. Tests automatically skip if devnet is unavailable
2. Use local validator for faster tests:
   ```bash
   solana-test-validator
   export SOLANA_TEST_RPC="http://localhost:8899"
   ```

### Race Detector Warnings

Run tests with race detector:
```bash
go test -race ./...
```

## Best Practices

1. **Use `t.Helper()`** for test utility functions
2. **Clean up resources** with `t.Cleanup()` or `defer`
3. **Skip appropriately**: Use `t.Skip()` for missing dependencies
4. **Parallel tests**: Use `t.Parallel()` for independent tests
5. **Table-driven tests**: For multiple similar scenarios
6. **Clear test names**: Use `TestFunction_Scenario` pattern
7. **Mock external services**: Use mocks for unit tests
8. **Test error cases**: Don't just test happy paths

## Debugging Tests

### Verbose Output
```bash
go test -v ./api -run TestSpecificTest
```

### Print Values
```bash
go test -v ./api 2>&1 | grep "test output"
```

### Run Single Test
```bash
go test ./vault -run TestEncryptAESGCM_Success -v
```

### Increase Timeout
```bash
go test -timeout 30s ./db
```

## Contributing

When adding new code:
1. Write tests first (TDD) or alongside implementation
2. Aim for >80% code coverage
3. Include both success and error cases
4. Add integration tests for database/external service interactions
5. Update this README if adding new test patterns or requirements
