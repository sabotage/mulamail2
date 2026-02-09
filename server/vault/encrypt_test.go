package vault

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"
)

// generateTestKey creates a valid 32-byte hex key for testing
func generateTestKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	return hex.EncodeToString(key)
}

func TestEncryptAESGCM_Success(t *testing.T) {
	key := generateTestKey(t)
	plaintext := "test secret password"

	ciphertext, err := EncryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptAESGCM failed: %v", err)
	}

	if ciphertext == "" {
		t.Fatal("expected non-empty ciphertext")
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(ciphertext); err != nil {
		t.Errorf("ciphertext is not valid hex: %v", err)
	}

	// Verify ciphertext is different from plaintext
	if ciphertext == plaintext {
		t.Error("ciphertext should not equal plaintext")
	}
}

func TestDecryptAESGCM_Success(t *testing.T) {
	key := generateTestKey(t)
	plaintext := "my-super-secret-password-123"

	ciphertext, err := EncryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := DecryptAESGCM(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptAESGCM failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted text doesn't match original.\nwant: %q\ngot:  %q", plaintext, decrypted)
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	testCases := []struct {
		name      string
		plaintext string
	}{
		{"empty string", ""},
		{"simple text", "hello"},
		{"special characters", "p@ssw0rd!#$%^&*()"},
		{"unicode", "こんにちは世界🌍"},
		{"multiline", "line1\nline2\nline3"},
		{"long text", strings.Repeat("a", 1000)},
		{"email credentials", "user@example.com:MyP@ssw0rd123!"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := generateTestKey(t)

			// Encrypt
			ciphertext, err := EncryptAESGCM(key, tc.plaintext)
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			// Decrypt
			decrypted, err := DecryptAESGCM(key, ciphertext)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			if decrypted != tc.plaintext {
				t.Errorf("round-trip failed.\noriginal: %q\ndecrypted: %q", tc.plaintext, decrypted)
			}
		})
	}
}

func TestEncryptAESGCM_DifferentNonces(t *testing.T) {
	key := generateTestKey(t)
	plaintext := "test"

	// Encrypt the same plaintext multiple times
	ct1, err := EncryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("first encryption failed: %v", err)
	}

	ct2, err := EncryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("second encryption failed: %v", err)
	}

	// Each encryption should produce different ciphertext due to random nonce
	if ct1 == ct2 {
		t.Error("encrypting same plaintext twice should produce different ciphertexts (different nonces)")
	}

	// Both should decrypt correctly
	pt1, _ := DecryptAESGCM(key, ct1)
	pt2, _ := DecryptAESGCM(key, ct2)
	if pt1 != plaintext || pt2 != plaintext {
		t.Error("both ciphertexts should decrypt to original plaintext")
	}
}

func TestEncryptAESGCM_InvalidKey(t *testing.T) {
	testCases := []struct {
		name string
		key  string
	}{
		{"invalid hex", "not-hex-at-all"},
		{"too short", "0123456789abcdef"},
		{"wrong length (31 bytes)", strings.Repeat("00", 31)},
		{"wrong length (33 bytes)", strings.Repeat("00", 33)},
		{"empty key", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EncryptAESGCM(tc.key, "test")
			if err == nil {
				t.Error("expected error with invalid key, got nil")
			}
		})
	}
}

func TestDecryptAESGCM_InvalidKey(t *testing.T) {
	// First create a valid ciphertext
	validKey := generateTestKey(t)
	ciphertext, _ := EncryptAESGCM(validKey, "test")

	// Try to decrypt with wrong key
	wrongKey := generateTestKey(t)
	_, err := DecryptAESGCM(wrongKey, ciphertext)
	if err == nil {
		t.Error("expected error decrypting with wrong key, got nil")
	}
}

func TestDecryptAESGCM_InvalidCiphertext(t *testing.T) {
	key := generateTestKey(t)

	testCases := []struct {
		name       string
		ciphertext string
	}{
		{"invalid hex", "not-valid-hex"},
		{"too short", "abcd"},
		{"empty", ""},
		{"corrupted", "0000000000000000000000000000"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecryptAESGCM(key, tc.ciphertext)
			if err == nil {
				t.Error("expected error with invalid ciphertext, got nil")
			}
		})
	}
}

func TestDecryptAESGCM_TamperedCiphertext(t *testing.T) {
	key := generateTestKey(t)
	plaintext := "important secret"

	ciphertext, err := EncryptAESGCM(key, plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Tamper with the ciphertext (flip a bit)
	ctBytes, _ := hex.DecodeString(ciphertext)
	if len(ctBytes) > 20 {
		ctBytes[20] ^= 0x01 // Flip one bit
	}
	tamperedCT := hex.EncodeToString(ctBytes)

	// Decryption should fail due to authentication tag mismatch
	_, err = DecryptAESGCM(key, tamperedCT)
	if err == nil {
		t.Error("expected error with tampered ciphertext (GCM auth should fail)")
	}
}

func TestEncryptDecrypt_MultipleKeys(t *testing.T) {
	plaintext := "test data"
	key1 := generateTestKey(t)
	key2 := generateTestKey(t)

	// Encrypt with key1
	ct1, err := EncryptAESGCM(key1, plaintext)
	if err != nil {
		t.Fatalf("encryption with key1 failed: %v", err)
	}

	// Should decrypt with key1
	pt1, err := DecryptAESGCM(key1, ct1)
	if err != nil {
		t.Fatalf("decryption with key1 failed: %v", err)
	}
	if pt1 != plaintext {
		t.Error("decryption with correct key failed")
	}

	// Should NOT decrypt with key2
	_, err = DecryptAESGCM(key2, ct1)
	if err == nil {
		t.Error("decryption with wrong key should fail")
	}
}

// Benchmark encryption performance
func BenchmarkEncryptAESGCM(b *testing.B) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	plaintext := "benchmark test password"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncryptAESGCM(key, plaintext)
	}
}

// Benchmark decryption performance
func BenchmarkDecryptAESGCM(b *testing.B) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	plaintext := "benchmark test password"
	ciphertext, _ := EncryptAESGCM(key, plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecryptAESGCM(key, ciphertext)
	}
}
