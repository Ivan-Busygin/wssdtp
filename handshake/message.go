// Package handshake provides data structures and utilities for WSSDTP handshake messages.
// The handshake message is a fixed-size 82-byte structure used for initial protocol negotiation.
package handshake

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
)

// HandshakeMessage represents the structure of a WSSDTP handshake message.
// It contains protocol version, cryptographic keys, authentication token, and random data.
// The total size is 82 bytes when marshaled.
type HandshakeMessage struct {
	Version   uint16    // Protocol version (major.minor encoded as uint16)
	Random    [32]byte  // Random data for key derivation salt
	PublicKey [32]byte  // X25519 public key
	AuthToken [16]byte  // Authentication token
}

// Marshal serializes the HandshakeMessage into a byte slice.
// The resulting slice is always 82 bytes long.
//
// Returns:
//   - []byte: The marshaled handshake message
func (m *HandshakeMessage) Marshal() []byte {
	buf := make([]byte, 82)
	binary.BigEndian.PutUint16(buf[0:2], m.Version)
	copy(buf[2:34], m.Random[:])
	copy(buf[34:66], m.PublicKey[:])
	copy(buf[66:82], m.AuthToken[:])
	return buf
}

// Unmarshal deserializes a byte slice into a HandshakeMessage.
// The input must be exactly 82 bytes long.
//
// Parameters:
//   - data: The byte slice to unmarshal (must be 82 bytes)
//
// Returns:
//   - error: An error if the data is invalid or wrong size
func (m *HandshakeMessage) Unmarshal(data []byte) error {
	if len(data) != 82 {
		return errors.New("handshake message must be 82 bytes")
	}
	m.Version = binary.BigEndian.Uint16(data[0:2])
	copy(m.Random[:], data[2:34])
	copy(m.PublicKey[:], data[34:66])
	copy(m.AuthToken[:], data[66:82])
	return nil
}

// GenerateRandom fills the Random field with cryptographically secure random bytes.
// This is used to generate unique salt data for each handshake.
//
// Returns:
//   - error: An error if random generation fails (extremely unlikely)
func (m *HandshakeMessage) GenerateRandom() error {
	if _, err := io.ReadFull(rand.Reader, m.Random[:]); err != nil {
		return err
	}
	return nil
}
