package vault

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage implements the Storage interface using local filesystem.
// Files are stored in a configurable directory with the key as the relative path.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a new local file storage.
// baseDir is where all files will be stored (e.g., "./data/vault")
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create base directory: %w", err)
	}

	// Verify directory is writable
	testFile := filepath.Join(baseDir, ".write-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return nil, fmt.Errorf("base directory not writable: %w", err)
	}
	os.Remove(testFile)

	return &LocalStorage{
		baseDir: baseDir,
	}, nil
}

// Put stores raw bytes at the given key (filepath).
func (l *LocalStorage) Put(ctx context.Context, key string, data []byte) error {
	// Sanitize key to prevent directory traversal
	key = filepath.Clean(key)
	if strings.Contains(key, "..") {
		return fmt.Errorf("invalid key: contains '..'")
	}

	fullPath := filepath.Join(l.baseDir, key)

	// Create parent directories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write file with secure permissions (owner read/write only)
	if err := os.WriteFile(fullPath, data, 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// Get retrieves the object at the given key (filepath).
func (l *LocalStorage) Get(ctx context.Context, key string) ([]byte, error) {
	// Sanitize key
	key = filepath.Clean(key)
	if strings.Contains(key, "..") {
		return nil, fmt.Errorf("invalid key: contains '..'")
	}

	fullPath := filepath.Join(l.baseDir, key)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("read file: %w", err)
	}

	return data, nil
}

// Delete removes the object at the given key.
func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	// Sanitize key
	key = filepath.Clean(key)
	if strings.Contains(key, "..") {
		return fmt.Errorf("invalid key: contains '..'")
	}

	fullPath := filepath.Join(l.baseDir, key)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted, consider it success
		}
		return fmt.Errorf("delete file: %w", err)
	}

	// Try to remove empty parent directories (optional cleanup)
	dir := filepath.Dir(fullPath)
	for dir != l.baseDir {
		if os.Remove(dir) != nil {
			break // Directory not empty or error, stop
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

// List returns all keys with the given prefix.
func (l *LocalStorage) List(ctx context.Context, prefix string) ([]string, error) {
	// Sanitize prefix
	prefix = filepath.Clean(prefix)
	if strings.Contains(prefix, "..") {
		return nil, fmt.Errorf("invalid prefix: contains '..'")
	}

	searchPath := filepath.Join(l.baseDir, prefix)
	var keys []string

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// If the prefix path doesn't exist, return empty list
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from baseDir
		relPath, err := filepath.Rel(l.baseDir, path)
		if err != nil {
			return err
		}

		keys = append(keys, relPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	return keys, nil
}

// BaseDir returns the base directory where files are stored.
func (l *LocalStorage) BaseDir() string {
	return l.baseDir
}
