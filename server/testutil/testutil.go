package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
)

// GenerateEncryptionKey creates a valid 32-byte hex key for testing
func GenerateEncryptionKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}
	return hex.EncodeToString(key)
}

// SetEnvForTest sets an environment variable for the duration of a test
func SetEnvForTest(t *testing.T, key, value string) {
	t.Helper()
	old := os.Getenv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if old == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, old)
		}
	})
}

// SkipIfShort skips a test if running in short mode
func SkipIfShort(t *testing.T, reason string) {
	t.Helper()
	if testing.Short() {
		t.Skipf("skipping in short mode: %s", reason)
	}
}

// MustHaveEnv skips the test if the given environment variable is not set
func MustHaveEnv(t *testing.T, key string) string {
	t.Helper()
	val := os.Getenv(key)
	if val == "" {
		t.Skipf("test requires %s environment variable", key)
	}
	return val
}

// RequireMongoURI returns the MongoDB test URI or skips the test
func RequireMongoURI(t *testing.T) string {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	return uri
}

// RequireSolanaRPC returns the Solana RPC URL or skips the test
func RequireSolanaRPC(t *testing.T) string {
	t.Helper()
	rpc := os.Getenv("SOLANA_TEST_RPC")
	if rpc == "" {
		rpc = "https://api.devnet.solana.com"
	}
	return rpc
}
