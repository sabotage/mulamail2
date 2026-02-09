package config

import "os"

// Config holds all runtime configuration, populated from environment variables.
type Config struct {
	Port          string
	MongoURI      string
	MongoDBName   string
	SolanaRPC     string
	StorageType   string // "local" or "s3"
	LocalDataPath string // Path for local storage (when StorageType=local)
	AWSRegion     string
	S3Bucket      string
	EncryptionKey string // hex-encoded 32-byte key for AES-256-GCM credential storage
}

func Load() *Config {
	return &Config{
		Port:          env("PORT", "8080"),
		MongoURI:      env("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName:   env("MONGO_DB", "mulamail"),
		SolanaRPC:     env("SOLANA_RPC", "https://api.mainnet-beta.solana.com"),
		StorageType:   env("STORAGE_TYPE", "local"),
		LocalDataPath: env("LOCAL_DATA_PATH", "./data/vault"),
		AWSRegion:     env("AWS_REGION", "us-east-1"),
		S3Bucket:      env("S3_BUCKET", "mulamail-vault"),
		EncryptionKey: env("ENCRYPTION_KEY", "0000000000000000000000000000000000000000000000000000000000000000"),
	}
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
