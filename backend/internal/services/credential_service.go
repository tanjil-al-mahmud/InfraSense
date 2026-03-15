package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// CredentialService handles encryption and decryption of credentials
type CredentialService struct {
	encryptionKey []byte
}

// NewCredentialService creates a new credential service
func NewCredentialService(encryptionKey string) (*CredentialService, error) {
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes, got %d", len(encryptionKey))
	}

	return &CredentialService{
		encryptionKey: []byte(encryptionKey),
	}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (s *CredentialService) Encrypt(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, fmt.Errorf("plaintext cannot be empty")
	}

	// Create AES cipher block
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random 96-bit nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (s *CredentialService) Decrypt(ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", fmt.Errorf("ciphertext cannot be empty")
	}

	// Create AES cipher block
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce from ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptToBase64 encrypts plaintext and returns base64-encoded ciphertext
func (s *CredentialService) EncryptToBase64(plaintext string) (string, error) {
	ciphertext, err := s.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptFromBase64 decrypts base64-encoded ciphertext
func (s *CredentialService) DecryptFromBase64(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	return s.Decrypt(ciphertext)
}
