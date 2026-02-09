package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear all relevant environment variables
	envVars := []string{
		"PORT", "MONGO_URI", "MONGO_DB", "SOLANA_RPC",
		"AWS_REGION", "S3_BUCKET", "ENCRYPTION_KEY",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	cfg := Load()

	expected := &Config{
		Port:          "8080",
		MongoURI:      "mongodb://localhost:27017",
		MongoDBName:   "mulamail",
		SolanaRPC:     "https://api.mainnet-beta.solana.com",
		AWSRegion:     "us-east-1",
		S3Bucket:      "mulamail-vault",
		EncryptionKey: "0000000000000000000000000000000000000000000000000000000000000000",
	}

	if cfg.Port != expected.Port {
		t.Errorf("Port: want %q, got %q", expected.Port, cfg.Port)
	}
	if cfg.MongoURI != expected.MongoURI {
		t.Errorf("MongoURI: want %q, got %q", expected.MongoURI, cfg.MongoURI)
	}
	if cfg.MongoDBName != expected.MongoDBName {
		t.Errorf("MongoDBName: want %q, got %q", expected.MongoDBName, cfg.MongoDBName)
	}
	if cfg.SolanaRPC != expected.SolanaRPC {
		t.Errorf("SolanaRPC: want %q, got %q", expected.SolanaRPC, cfg.SolanaRPC)
	}
	if cfg.AWSRegion != expected.AWSRegion {
		t.Errorf("AWSRegion: want %q, got %q", expected.AWSRegion, cfg.AWSRegion)
	}
	if cfg.S3Bucket != expected.S3Bucket {
		t.Errorf("S3Bucket: want %q, got %q", expected.S3Bucket, cfg.S3Bucket)
	}
	if cfg.EncryptionKey != expected.EncryptionKey {
		t.Errorf("EncryptionKey: want %q, got %q", expected.EncryptionKey, cfg.EncryptionKey)
	}
}

func TestLoad_CustomEnvironmentVariables(t *testing.T) {
	// Set custom environment variables
	testEnv := map[string]string{
		"PORT":           "3000",
		"MONGO_URI":      "mongodb://testhost:27017",
		"MONGO_DB":       "testdb",
		"SOLANA_RPC":     "https://api.devnet.solana.com",
		"AWS_REGION":     "us-west-2",
		"S3_BUCKET":      "test-bucket",
		"ENCRYPTION_KEY": "1111111111111111111111111111111111111111111111111111111111111111",
	}

	// Set env vars
	for k, v := range testEnv {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("failed to set env var %s: %v", k, err)
		}
	}

	// Clean up after test
	defer func() {
		for k := range testEnv {
			os.Unsetenv(k)
		}
	}()

	cfg := Load()

	if cfg.Port != testEnv["PORT"] {
		t.Errorf("Port: want %q, got %q", testEnv["PORT"], cfg.Port)
	}
	if cfg.MongoURI != testEnv["MONGO_URI"] {
		t.Errorf("MongoURI: want %q, got %q", testEnv["MONGO_URI"], cfg.MongoURI)
	}
	if cfg.MongoDBName != testEnv["MONGO_DB"] {
		t.Errorf("MongoDBName: want %q, got %q", testEnv["MONGO_DB"], cfg.MongoDBName)
	}
	if cfg.SolanaRPC != testEnv["SOLANA_RPC"] {
		t.Errorf("SolanaRPC: want %q, got %q", testEnv["SOLANA_RPC"], cfg.SolanaRPC)
	}
	if cfg.AWSRegion != testEnv["AWS_REGION"] {
		t.Errorf("AWSRegion: want %q, got %q", testEnv["AWS_REGION"], cfg.AWSRegion)
	}
	if cfg.S3Bucket != testEnv["S3_BUCKET"] {
		t.Errorf("S3Bucket: want %q, got %q", testEnv["S3_BUCKET"], cfg.S3Bucket)
	}
	if cfg.EncryptionKey != testEnv["ENCRYPTION_KEY"] {
		t.Errorf("EncryptionKey: want %q, got %q", testEnv["ENCRYPTION_KEY"], cfg.EncryptionKey)
	}
}

func TestLoad_PartialEnvironmentVariables(t *testing.T) {
	// Clear all
	envVars := []string{
		"PORT", "MONGO_URI", "MONGO_DB", "SOLANA_RPC",
		"AWS_REGION", "S3_BUCKET", "ENCRYPTION_KEY",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	// Set only some variables
	os.Setenv("PORT", "9000")
	os.Setenv("SOLANA_RPC", "https://api.testnet.solana.com")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("SOLANA_RPC")
	}()

	cfg := Load()

	// Should use custom values
	if cfg.Port != "9000" {
		t.Errorf("Port: want %q, got %q", "9000", cfg.Port)
	}
	if cfg.SolanaRPC != "https://api.testnet.solana.com" {
		t.Errorf("SolanaRPC: want %q, got %q", "https://api.testnet.solana.com", cfg.SolanaRPC)
	}

	// Should use defaults for unset vars
	if cfg.MongoURI != "mongodb://localhost:27017" {
		t.Errorf("MongoURI should use default, got %q", cfg.MongoURI)
	}
	if cfg.AWSRegion != "us-east-1" {
		t.Errorf("AWSRegion should use default, got %q", cfg.AWSRegion)
	}
}

func TestLoad_EmptyEnvironmentVariables(t *testing.T) {
	// Set environment variables to empty strings
	os.Setenv("PORT", "")
	os.Setenv("MONGO_URI", "")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("MONGO_URI")
	}()

	cfg := Load()

	// Empty env vars should be used (not fallback to defaults)
	if cfg.Port != "" {
		t.Errorf("Port: want empty string, got %q", cfg.Port)
	}
	if cfg.MongoURI != "" {
		t.Errorf("MongoURI: want empty string, got %q", cfg.MongoURI)
	}
}

func TestEnv_Function(t *testing.T) {
	testCases := []struct {
		name     string
		key      string
		fallback string
		setValue string
		setEnv   bool
		expected string
	}{
		{
			name:     "env var not set, use fallback",
			key:      "TEST_VAR_1",
			fallback: "default",
			setEnv:   false,
			expected: "default",
		},
		{
			name:     "env var set, use value",
			key:      "TEST_VAR_2",
			fallback: "default",
			setValue: "custom",
			setEnv:   true,
			expected: "custom",
		},
		{
			name:     "env var set to empty string",
			key:      "TEST_VAR_3",
			fallback: "default",
			setValue: "",
			setEnv:   true,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up
			os.Unsetenv(tc.key)
			defer os.Unsetenv(tc.key)

			if tc.setEnv {
				os.Setenv(tc.key, tc.setValue)
			}

			result := env(tc.key, tc.fallback)
			if result != tc.expected {
				t.Errorf("env(%q, %q): want %q, got %q", tc.key, tc.fallback, tc.expected, result)
			}
		})
	}
}

func TestLoad_ProductionLikeConfig(t *testing.T) {
	// Simulate a production configuration
	prodEnv := map[string]string{
		"PORT":           "443",
		"MONGO_URI":      "mongodb+srv://user:pass@cluster.mongodb.net/",
		"MONGO_DB":       "mulamail_prod",
		"SOLANA_RPC":     "https://api.mainnet-beta.solana.com",
		"AWS_REGION":     "eu-west-1",
		"S3_BUCKET":      "mulamail-prod-vault",
		"ENCRYPTION_KEY": "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
	}

	for k, v := range prodEnv {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range prodEnv {
			os.Unsetenv(k)
		}
	}()

	cfg := Load()

	// Verify all production values are loaded correctly
	if cfg.Port != prodEnv["PORT"] {
		t.Errorf("Port mismatch in prod config")
	}
	if cfg.MongoURI != prodEnv["MONGO_URI"] {
		t.Errorf("MongoURI mismatch in prod config")
	}
	if cfg.MongoDBName != prodEnv["MONGO_DB"] {
		t.Errorf("MongoDBName mismatch in prod config")
	}
	if cfg.SolanaRPC != prodEnv["SOLANA_RPC"] {
		t.Errorf("SolanaRPC mismatch in prod config")
	}
	if cfg.AWSRegion != prodEnv["AWS_REGION"] {
		t.Errorf("AWSRegion mismatch in prod config")
	}
	if cfg.S3Bucket != prodEnv["S3_BUCKET"] {
		t.Errorf("S3Bucket mismatch in prod config")
	}
	if cfg.EncryptionKey != prodEnv["ENCRYPTION_KEY"] {
		t.Errorf("EncryptionKey mismatch in prod config")
	}
}

func TestLoad_DevnetConfig(t *testing.T) {
	// Simulate a devnet/test configuration
	os.Setenv("SOLANA_RPC", "https://api.devnet.solana.com")
	os.Setenv("MONGO_DB", "mulamail_dev")
	defer func() {
		os.Unsetenv("SOLANA_RPC")
		os.Unsetenv("MONGO_DB")
	}()

	cfg := Load()

	if cfg.SolanaRPC != "https://api.devnet.solana.com" {
		t.Errorf("expected devnet RPC URL")
	}
	if cfg.MongoDBName != "mulamail_dev" {
		t.Errorf("expected dev database name")
	}
}
