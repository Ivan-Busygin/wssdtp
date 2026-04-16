package transport

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
)

// TLSTransport implements Transport interface over raw TLS connection
type TLSTransport struct {
	conn   *tls.Conn
	reader *bufio.Reader
}

// NewTLSClientTransport creates a new TLS transport for client
func NewTLSClientTransport(addr string, tlsConfig *tls.Config) (*TLSTransport, error) {
	if tlsConfig == nil {
		return nil, errors.New("TLS config is required")
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	return &TLSTransport{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

// NewTLSServerTransport creates a new TLS transport for server
func NewTLSServerTransport(conn *tls.Conn) *TLSTransport {
	return &TLSTransport{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

// ReadMessage reads a message from the TLS connection.
// Messages are framed with a 4-byte big-endian length prefix.
func (t *TLSTransport) ReadMessage() ([]byte, error) {
	// Read 4-byte length prefix
	var length uint32
	if err := binary.Read(t.reader, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// Validate length (prevent DoS with huge allocations)
	if length > 10*1024*1024 { // 10MB limit
		return nil, errors.New("message too large")
	}

	// Read the message data
	data := make([]byte, length)
	if _, err := io.ReadFull(t.reader, data); err != nil {
		return nil, err
	}

	return data, nil
}

// WriteMessage writes a message to the TLS connection.
// Messages are framed with a 4-byte big-endian length prefix.
func (t *TLSTransport) WriteMessage(data []byte) error {
	// Check message size
	if len(data) > 10*1024*1024 {
		return errors.New("message too large")
	}

	// Write length prefix
	if err := binary.Write(t.conn, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}

	// Write message data
	_, err := t.conn.Write(data)
	return err
}

// Close closes the TLS connection
func (t *TLSTransport) Close() error {
	return t.conn.Close()
}

// GetTLSConnection returns the underlying TLS connection for advanced operations
func (t *TLSTransport) GetTLSConnection() *tls.Conn {
	return t.conn
}