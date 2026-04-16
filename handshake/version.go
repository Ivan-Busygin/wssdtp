package handshake

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// ProtocolVersionMajor is the major version number of the protocol
	ProtocolVersionMajor uint8 = 0

	// ProtocolVersionMinor is the minor version number of the protocol
	ProtocolVersionMinor uint8 = 0

	// ProtocolVersionString is the human-readable version string
	ProtocolVersionString = "0.0"
)

// EncodeVersion encodes major and minor version numbers into a uint16.
// The major version is stored in the high byte, minor version in the low byte.
// For example: EncodeVersion(1, 0) returns 0x0100 = 256
func EncodeVersion(major, minor uint8) uint16 {
	return (uint16(major) << 8) | uint16(minor)
}

// DecodeVersion decodes a uint16 into major and minor version numbers.
// The high byte represents major version, low byte represents minor version.
// For example: DecodeVersion(0x0100) returns (1, 0)
func DecodeVersion(v uint16) (major, minor uint8) {
	major = uint8(v >> 8)
	minor = uint8(v & 0xFF)
	return
}

// ParseVersionString parses a version string like "1.0" into major and minor components.
// Returns the major and minor version numbers or an error if the format is invalid.
func ParseVersionString(s string) (major, minor uint8, err error) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid version format: expected 'major.minor', got '%s'", s)
	}

	majInt, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major version: %v", err)
	}

	minInt, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minor version: %v", err)
	}

	return uint8(majInt), uint8(minInt), nil
}

// VersionString returns the human-readable version string from major and minor components.
func VersionString(major, minor uint8) string {
	return fmt.Sprintf("%d.%d", major, minor)
}
