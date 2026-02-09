package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// EncryptAESGCM encrypts plaintext with AES-256-GCM.
// key must be a hex-encoded 32-byte value (64 hex characters).
// Returns the nonce+ciphertext as a hex string.
func EncryptAESGCM(key, plaintext string) (string, error) {
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("decode encryption key: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Seal appends ciphertext to nonce so the output is nonce||ciphertext.
	out := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(out), nil
}

// DecryptAESGCM is the inverse of EncryptAESGCM.
func DecryptAESGCM(key, ciphertextHex string) (string, error) {
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("decode encryption key: %w", err)
	}

	data, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
