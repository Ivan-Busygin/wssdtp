package frame_test

import (
	"bytes"
	"testing"

	"wssdtp/frame"
)

func TestEncodeDecodeHeader(t *testing.T) {
	tests := []struct {
		typ      byte
		streamID uint16
		length   uint16
	}{
		{frame.TypeData, 0, 0},
		{frame.TypeData, 1, 100},
		{frame.TypeOpen, 42, 1024},
		{frame.TypeClose, 65535, 65535},
		{frame.TypePing, 123, 0},
		{frame.TypePong, 456, 0},
	}

	for _, test := range tests {
		encoded := frame.EncodeHeader(test.typ, test.streamID, test.length)
		if len(encoded) != 5 {
			t.Errorf("Header length: expected 5, got %d", len(encoded))
		}

		decoded, err := frame.DecodeHeader(encoded)
		if err != nil {
			t.Fatalf("DecodeHeader failed: %v", err)
		}

		if decoded.Type != test.typ {
			t.Errorf("Type mismatch: expected %d, got %d", test.typ, decoded.Type)
		}
		if decoded.StreamID != test.streamID {
			t.Errorf("StreamID mismatch: expected %d, got %d", test.streamID, decoded.StreamID)
		}
		if decoded.Length != test.length {
			t.Errorf("Length mismatch: expected %d, got %d", test.length, decoded.Length)
		}
	}
}

func TestDecodeHeaderInvalidLength(t *testing.T) {
	// Test with too short data
	_, err := frame.DecodeHeader([]byte{1, 2, 3})
	if err == nil {
		t.Error("Expected error for too short header, got nil")
	}
}

func TestBuildPayload(t *testing.T) {
	plaintext := []byte("Hello, World!")
	maxPadding := 100

	payload := frame.BuildPayload(plaintext, maxPadding)

	// Check minimum size: 2 bytes length + plaintext + at least 0 padding
	minExpectedSize := 2 + len(plaintext)
	if len(payload) < minExpectedSize {
		t.Errorf("Payload too small: expected at least %d, got %d", minExpectedSize, len(payload))
	}

	// Extract real length
	if len(payload) < 2 {
		t.Fatal("Payload too short to contain length")
	}
	realLen := uint16(payload[0])<<8 | uint16(payload[1])
	if int(realLen) != len(plaintext) {
		t.Errorf("Real length mismatch: expected %d, got %d", len(plaintext), realLen)
	}

	// Check that real data matches
	actualData := payload[2 : 2+realLen]
	if !bytes.Equal(actualData, plaintext) {
		t.Errorf("Data mismatch: expected %q, got %q", plaintext, actualData)
	}
}

func TestBuildPayloadNoPadding(t *testing.T) {
	plaintext := []byte("Test")
	payload := frame.BuildPayload(plaintext, 0) // No padding

	expectedSize := 2 + len(plaintext) // 2 bytes length + data
	if len(payload) != expectedSize {
		t.Errorf("Payload size with no padding: expected %d, got %d", expectedSize, len(payload))
	}
}

func TestExtractPayload(t *testing.T) {
	plaintext := []byte("Test data")
	payload := frame.BuildPayload(plaintext, 50)

	extracted, err := frame.ExtractPayload(payload)
	if err != nil {
		t.Fatalf("ExtractPayload failed: %v", err)
	}

	if !bytes.Equal(extracted, plaintext) {
		t.Errorf("Extracted data mismatch: expected %q, got %q", plaintext, extracted)
	}
}

func TestExtractPayloadInvalid(t *testing.T) {
	// Test with too short payload
	_, err := frame.ExtractPayload([]byte{1})
	if err == nil {
		t.Error("Expected error for too short payload, got nil")
	}

	// Test with length claiming more data than available
	payload := []byte{0, 10, 1, 2, 3} // Claims 10 bytes but only 3 available
	_, err = frame.ExtractPayload(payload)
	if err == nil {
		t.Error("Expected error for invalid length, got nil")
	}
}

func TestFrameTypes(t *testing.T) {
	expectedTypes := map[byte]string{
		frame.TypeData:  "Data",
		frame.TypeOpen:  "Open",
		frame.TypeClose: "Close",
		frame.TypePing:  "Ping",
		frame.TypePong:  "Pong",
	}

	for typ, name := range expectedTypes {
		if typ < 0x01 || typ > 0x05 {
			t.Errorf("Frame type %s (%d) out of expected range [0x01, 0x05]", name, typ)
		}
	}
}