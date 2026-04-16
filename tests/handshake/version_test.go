package handshake_test

import (
	"testing"

	"wssdtp/handshake"
)

func TestEncodeVersion(t *testing.T) {
	tests := []struct {
		major    uint8
		minor    uint8
		expected uint16
	}{
		{0, 0, 0x0000},
		{0, 1, 0x0001},
		{1, 0, 0x0100},
		{1, 1, 0x0101},
		{255, 255, 0xFFFF},
	}

	for _, test := range tests {
		result := handshake.EncodeVersion(test.major, test.minor)
		if result != test.expected {
			t.Errorf("EncodeVersion(%d, %d) = %d, expected %d",
				test.major, test.minor, result, test.expected)
		}
	}
}

func TestDecodeVersion(t *testing.T) {
	tests := []struct {
		input    uint16
		major    uint8
		minor    uint8
	}{
		{0x0000, 0, 0},
		{0x0001, 0, 1},
		{0x0100, 1, 0},
		{0x0101, 1, 1},
		{0xFFFF, 255, 255},
	}

	for _, test := range tests {
		major, minor := handshake.DecodeVersion(test.input)
		if major != test.major || minor != test.minor {
			t.Errorf("DecodeVersion(%d) = (%d, %d), expected (%d, %d)",
				test.input, major, minor, test.major, test.minor)
		}
	}
}

func TestEncodeDecodeVersionRoundTrip(t *testing.T) {
	// Test that encoding and decoding are inverses
	originalMajor := uint8(42)
	originalMinor := uint8(17)

	encoded := handshake.EncodeVersion(originalMajor, originalMinor)
	decodedMajor, decodedMinor := handshake.DecodeVersion(encoded)

	if decodedMajor != originalMajor || decodedMinor != originalMinor {
		t.Errorf("Round trip failed: (%d, %d) -> %d -> (%d, %d)",
			originalMajor, originalMinor, encoded, decodedMajor, decodedMinor)
	}
}

func TestParseVersionString(t *testing.T) {
	tests := []struct {
		input    string
		major    uint8
		minor    uint8
		hasError bool
	}{
		{"0.0", 0, 0, false},
		{"1.0", 1, 0, false},
		{"1.1", 1, 1, false},
		{"255.255", 255, 255, false},
		{"1", 0, 0, true},        // Missing minor
		{"1.2.3", 0, 0, true},    // Too many parts
		{"a.b", 0, 0, true},      // Non-numeric
		{"", 0, 0, true},         // Empty
		{"1.", 0, 0, true},       // Empty minor
		{".1", 0, 0, true},       // Empty major
	}

	for _, test := range tests {
		major, minor, err := handshake.ParseVersionString(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("ParseVersionString(%q) expected error, got nil", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseVersionString(%q) unexpected error: %v", test.input, err)
			}
			if major != test.major || minor != test.minor {
				t.Errorf("ParseVersionString(%q) = (%d, %d), expected (%d, %d)",
					test.input, major, minor, test.major, test.minor)
			}
		}
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		major   uint8
		minor   uint8
		expected string
	}{
		{0, 0, "0.0"},
		{1, 0, "1.0"},
		{1, 1, "1.1"},
		{255, 255, "255.255"},
	}

	for _, test := range tests {
		result := handshake.VersionString(test.major, test.minor)
		if result != test.expected {
			t.Errorf("VersionString(%d, %d) = %q, expected %q",
				test.major, test.minor, result, test.expected)
		}
	}
}

func TestProtocolVersionConstants(t *testing.T) {
	// Test that the constants are consistent
	expectedString := handshake.VersionString(handshake.ProtocolVersionMajor, handshake.ProtocolVersionMinor)
	if handshake.ProtocolVersionString != expectedString {
		t.Errorf("ProtocolVersionString inconsistency: %q != VersionString(%d, %d) = %q",
			handshake.ProtocolVersionString, handshake.ProtocolVersionMajor, handshake.ProtocolVersionMinor, expectedString)
	}

	// Test that the encoded version matches
	encoded := handshake.EncodeVersion(handshake.ProtocolVersionMajor, handshake.ProtocolVersionMinor)
	if encoded != 0x0000 { // Should be 0.0 = 0x0000
		t.Errorf("Protocol version encoding: expected 0x0000, got 0x%04x", encoded)
	}
}