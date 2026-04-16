package crypto_test

import (
	"crypto/rand"
	"testing"

	"wssdtp/crypto"
)

func TestGenerateKeyPair(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if len(priv) != 32 {
		t.Errorf("Private key length: expected 32, got %d", len(priv))
	}

	if len(pub) != 32 {
		t.Errorf("Public key length: expected 32, got %d", len(pub))
	}

	// Test that keys are not all zeros
	privAllZeros := true
	for _, b := range priv {
		if b != 0 {
			privAllZeros = false
			break
		}
	}
	if privAllZeros {
		t.Error("Private key appears to be all zeros")
	}

	pubAllZeros := true
	for _, b := range pub {
		if b != 0 {
			pubAllZeros = false
			break
		}
	}
	if pubAllZeros {
		t.Error("Public key appears to be all zeros")
	}
}

func TestComputeSharedSecret(t *testing.T) {
	// Generate two key pairs
	priv1, pub1, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate first key pair: %v", err)
	}

	priv2, pub2, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate second key pair: %v", err)
	}

	// Compute shared secrets from both sides
	shared1, err := crypto.ComputeSharedSecret(priv1, pub2)
	if err != nil {
		t.Fatalf("Failed to compute shared secret 1: %v", err)
	}

	shared2, err := crypto.ComputeSharedSecret(priv2, pub1)
	if err != nil {
		t.Fatalf("Failed to compute shared secret 2: %v", err)
	}

	if len(shared1) != 32 {
		t.Errorf("Shared secret 1 length: expected 32, got %d", len(shared1))
	}

	if len(shared2) != 32 {
		t.Errorf("Shared secret 2 length: expected 32, got %d", len(shared2))
	}

	// Shared secrets should be identical
	if string(shared1) != string(shared2) {
		t.Error("Shared secrets are not identical")
	}

	// Shared secret should not be all zeros
	allZeros := true
	for _, b := range shared1 {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Shared secret appears to be all zeros")
	}
}

func TestHKDFExpand(t *testing.T) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatalf("Failed to generate random secret: %v", err)
	}

	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		t.Fatalf("Failed to generate random salt: %v", err)
	}

	info := []byte("test-info")

	key, err := crypto.HKDFExpand(secret, salt, info, 32)
	if err != nil {
		t.Fatalf("HKDFExpand failed: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("Key length: expected 32, got %d", len(key))
	}

	// Key should not be all zeros
	allZeros := true
	for _, b := range key {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("HKDF key appears to be all zeros")
	}
}

func TestHKDFExpandDifferentLengths(t *testing.T) {
	secret := []byte("test-secret-12345678901234567890")
	salt := []byte("test-salt-1234567890")
	info := []byte("test-info")

	// Test different key lengths
	for length := 16; length <= 64; length += 8 {
		key, err := crypto.HKDFExpand(secret, salt, info, length)
		if err != nil {
			t.Fatalf("HKDFExpand failed for length %d: %v", length, err)
		}

		if len(key) != length {
			t.Errorf("Key length for %d: expected %d, got %d", length, length, len(key))
		}
	}
}

func TestNewAEAD(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	aead, err := crypto.NewAEAD(key)
	if err != nil {
		t.Fatalf("NewAEAD failed: %v", err)
	}

	if aead == nil {
		t.Error("NewAEAD returned nil AEAD")
	}

	// Test nonce size
	nonceSize := aead.NonceSize()
	if nonceSize != 12 { // ChaCha20-Poly1305 nonce size
		t.Errorf("Nonce size: expected 12, got %d", nonceSize)
	}

	// Test overhead
	overhead := aead.Overhead()
	if overhead != 16 { // ChaCha20-Poly1305 tag size
		t.Errorf("Overhead: expected 16, got %d", overhead)
	}
}

func TestNewAEADInvalidKey(t *testing.T) {
	// Test with invalid key lengths
	invalidKeys := [][]byte{
		make([]byte, 16), // Too short
		make([]byte, 24), // Wrong size
		make([]byte, 40), // Too long
	}

	for _, key := range invalidKeys {
		_, err := crypto.NewAEAD(key)
		if err == nil {
			t.Errorf("NewAEAD should fail with key length %d", len(key))
		}
	}
}