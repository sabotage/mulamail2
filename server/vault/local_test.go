package vault

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLocalStorage_Success(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-storage")
	defer os.RemoveAll(tmpDir)

	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage failed: %v", err)
	}

	if storage.baseDir != tmpDir {
		t.Errorf("baseDir: want %q, got %q", tmpDir, storage.baseDir)
	}

	// Verify directory exists
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("base directory was not created")
	}
}

func TestLocalStorage_PutGet_Success(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-put-get")
	defer os.RemoveAll(tmpDir)

	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage failed: %v", err)
	}

	ctx := context.Background()
	key := "test/file.txt"
	data := []byte("Hello, World!")

	// Put
	if err := storage.Put(ctx, key, data); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Get
	retrieved, err := storage.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(data) {
		t.Errorf("data mismatch: want %q, got %q", string(data), string(retrieved))
	}
}

func TestLocalStorage_GetNotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-not-found")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewLocalStorage(tmpDir)
	ctx := context.Background()

	_, err := storage.Get(ctx, "nonexistent.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestLocalStorage_Delete_Success(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-delete")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewLocalStorage(tmpDir)
	ctx := context.Background()

	key := "to-delete.txt"
	data := []byte("delete me")

	// Put
	storage.Put(ctx, key, data)

	// Verify exists
	if _, err := storage.Get(ctx, key); err != nil {
		t.Fatal("file should exist before delete")
	}

	// Delete
	if err := storage.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	if _, err := storage.Get(ctx, key); err == nil {
		t.Error("file should not exist after delete")
	}
}

func TestLocalStorage_Delete_NotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-delete-notfound")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewLocalStorage(tmpDir)
	ctx := context.Background()

	// Deleting non-existent file should not error
	if err := storage.Delete(ctx, "nonexistent.txt"); err != nil {
		t.Errorf("Delete of non-existent file should not error: %v", err)
	}
}

func TestLocalStorage_List_Success(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-list")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewLocalStorage(tmpDir)
	ctx := context.Background()

	// Create multiple files
	files := map[string][]byte{
		"dir1/file1.txt": []byte("content1"),
		"dir1/file2.txt": []byte("content2"),
		"dir2/file3.txt": []byte("content3"),
		"root.txt":       []byte("root"),
	}

	for key, data := range files {
		storage.Put(ctx, key, data)
	}

	// List all
	keys, err := storage.List(ctx, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(keys) != 4 {
		t.Errorf("expected 4 files, got %d", len(keys))
	}

	// List with prefix
	keysDir1, err := storage.List(ctx, "dir1")
	if err != nil {
		t.Fatalf("List with prefix failed: %v", err)
	}

	if len(keysDir1) != 2 {
		t.Errorf("expected 2 files in dir1, got %d", len(keysDir1))
	}
}

func TestLocalStorage_List_EmptyPrefix(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-list-empty")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewLocalStorage(tmpDir)
	ctx := context.Background()

	// List empty directory
	keys, err := storage.List(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("expected 0 files, got %d", len(keys))
	}
}

func TestLocalStorage_SecurityPathTraversal(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-security")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewLocalStorage(tmpDir)
	ctx := context.Background()

	maliciousKeys := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"subdir/../../escape",
	}

	for _, key := range maliciousKeys {
		// Put should reject
		err := storage.Put(ctx, key, []byte("malicious"))
		if err == nil {
			t.Errorf("Put should reject path traversal: %q", key)
		}

		// Get should reject
		_, err = storage.Get(ctx, key)
		if err == nil {
			t.Errorf("Get should reject path traversal: %q", key)
		}

		// Delete should reject
		err = storage.Delete(ctx, key)
		if err == nil {
			t.Errorf("Delete should reject path traversal: %q", key)
		}
	}
}

func TestLocalStorage_NestedDirectories(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-nested")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewLocalStorage(tmpDir)
	ctx := context.Background()

	key := "a/b/c/d/e/file.txt"
	data := []byte("deeply nested")

	// Put should create all parent directories
	if err := storage.Put(ctx, key, data); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify file exists
	retrieved, err := storage.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(data) {
		t.Error("data mismatch for nested file")
	}

	// Verify directory structure exists
	fullPath := filepath.Join(tmpDir, "a", "b", "c", "d", "e", "file.txt")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("nested directory structure was not created")
	}
}

func TestLocalStorage_FilePermissions(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-perms")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewLocalStorage(tmpDir)
	ctx := context.Background()

	key := "secure.txt"
	data := []byte("sensitive data")

	if err := storage.Put(ctx, key, data); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	fullPath := filepath.Join(tmpDir, key)
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// Check file has secure permissions (0600 = owner read/write only)
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("file permissions: want 0600, got %o", mode)
	}
}

func TestLocalStorage_BinaryData(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-binary")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewLocalStorage(tmpDir)
	ctx := context.Background()

	// Binary data with null bytes and special characters
	data := []byte{0x00, 0x01, 0xFF, 0xFE, 0x42, 0x00, 0x7F}
	key := "binary.dat"

	if err := storage.Put(ctx, key, data); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, err := storage.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(retrieved) != len(data) {
		t.Errorf("length mismatch: want %d, got %d", len(data), len(retrieved))
	}

	for i, b := range data {
		if retrieved[i] != b {
			t.Errorf("byte %d: want 0x%02x, got 0x%02x", i, b, retrieved[i])
		}
	}
}

func TestLocalStorage_LargeFile(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "mulamail-test-large")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewLocalStorage(tmpDir)
	ctx := context.Background()

	// Create 1MB of data
	size := 1024 * 1024
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	key := "large.bin"

	if err := storage.Put(ctx, key, data); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, err := storage.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(retrieved) != size {
		t.Errorf("size mismatch: want %d, got %d", size, len(retrieved))
	}
}
