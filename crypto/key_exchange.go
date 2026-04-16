package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

func GenerateKeyPair() (priv, pub []byte, err error) {
	priv = make([]byte, curve25519.ScalarSize)
	if _, err := io.ReadFull(rand.Reader, priv); err != nil {
		return nil, nil, err
	}
	pub, err = curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

func ComputeSharedSecret(private, peerPublic []byte) ([]byte, error) {
	return curve25519.X25519(private, peerPublic)
}

func HKDFExpand(secret, salt, info []byte, length int) ([]byte, error) {
	hkdf := hkdf.New(sha256.New, secret, salt, info)
	key := make([]byte, length)
	if _, err := io.ReadFull(hkdf, key); err != nil {
		return nil, err
	}
	return key, nil
}

func NewAEAD(key []byte) (cipher.AEAD, error) {
	return chacha20poly1305.New(key)
}
