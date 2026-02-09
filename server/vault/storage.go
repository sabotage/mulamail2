package vault

import "context"

// Storage defines the interface for storing encrypted mail data.
// Implementations include local file storage and cloud storage (S3, etc.).
type Storage interface {
	// Put stores raw bytes at the given key
	Put(ctx context.Context, key string, data []byte) error

	// Get retrieves the object at the given key
	Get(ctx context.Context, key string) ([]byte, error)

	// Delete removes the object at the given key (optional, can return nil if not implemented)
	Delete(ctx context.Context, key string) error

	// List returns all keys with the given prefix (optional, can return empty if not implemented)
	List(ctx context.Context, prefix string) ([]string, error)
}

// Ensure S3Client implements Storage interface
var _ Storage = (*S3Client)(nil)
var _ Storage = (*LocalStorage)(nil)
