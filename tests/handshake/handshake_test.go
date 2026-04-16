package handshake_test

import (
	"bytes"
	"testing"

	"wssdtp/handshake"
)

// mockTransport implements transport.Transport using a bytes.Buffer
type mockTransport struct {
	buf *bytes.Buffer
}

func (m *mockTransport) ReadMessage() ([]byte, error) {
	// For handshake, assume the message is exactly what's in the buffer
	data := make([]byte, m.buf.Len())
	n, err := m.buf.Read(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *mockTransport) WriteMessage(data []byte) error {
	_, err := m.buf.Write(data)
	return err
}

func (m *mockTransport) Close() error {
	return nil
}

func TestHandshakeMessageMarshalUnmarshal(t *testing.T) {
	// Create a test message
	original := handshake.HandshakeMessage{
		Version:   handshake.EncodeVersion(0, 0),
		Random:    [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		PublicKey: [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		AuthToken: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}

	// Marshal
	data := original.Marshal()
	if len(data) != 82 {
		t.Errorf("Expected marshaled data to be 82 bytes, got %d", len(data))
	}

	// Unmarshal
	var unmarshaled handshake.HandshakeMessage
	err := unmarshaled.Unmarshal(data)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Compare
	if unmarshaled.Version != original.Version {
		t.Errorf("Version mismatch: expected %d, got %d", original.Version, unmarshaled.Version)
	}
	if unmarshaled.Random != original.Random {
		t.Errorf("Random mismatch: expected %v, got %v", original.Random, unmarshaled.Random)
	}
	if unmarshaled.PublicKey != original.PublicKey {
		t.Errorf("PublicKey mismatch: expected %v, got %v", original.PublicKey, unmarshaled.PublicKey)
	}
	if unmarshaled.AuthToken != original.AuthToken {
		t.Errorf("AuthToken mismatch: expected %v, got %v", original.AuthToken, unmarshaled.AuthToken)
	}
}

func TestHandshakeMessageUnmarshalInvalidSize(t *testing.T) {
	var msg handshake.HandshakeMessage

	// Test with too short data
	err := msg.Unmarshal([]byte{1, 2, 3})
	if err == nil {
		t.Error("Expected error for too short data, got nil")
	}

	// Test with too long data
	err = msg.Unmarshal(make([]byte, 83))
	if err == nil {
		t.Error("Expected error for too long data, got nil")
	}
}

func TestHandshakeMessageGenerateRandom(t *testing.T) {
	var msg handshake.HandshakeMessage

	// Generate random data
	err := msg.GenerateRandom()
	if err != nil {
		t.Fatalf("Failed to generate random: %v", err)
	}

	// Check that random data is not all zeros
	allZeros := true
	for _, b := range msg.Random {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Random data appears to be all zeros, which is unlikely")
	}
}

func TestPerformClientHandshakeInvalidToken(t *testing.T) {
	// Test with invalid token length
	buf := &bytes.Buffer{}
	tr := &mockTransport{buf: buf}
	_, err := handshake.PerformClientHandshake(tr, []byte{1, 2, 3}, nil) // Too short
	if err == nil {
		t.Error("Expected error for invalid token length, got nil")
	}
}

func TestPerformServerHandshakeInvalidToken(t *testing.T) {
	// Test with invalid token
	buf := &bytes.Buffer{}
	tr := &mockTransport{buf: buf}
	_, err := handshake.PerformServerHandshake(tr, [][]byte{{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}, nil)
	if err == nil {
		t.Error("Expected error for invalid handshake data, got nil")
	}
}