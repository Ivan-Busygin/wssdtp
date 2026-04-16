package transport

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

// TCPTransport implements Transport interface over raw TCP connection
type TCPTransport struct {
	conn   net.Conn
	reader *bufio.Reader
}

// NewTCPClientTransport creates a new TCP transport for client
func NewTCPClientTransport(addr string, tlsConfig *tls.Config) (*TCPTransport, error) {
	var conn net.Conn
	var err error

	if tlsConfig != nil {
		conn, err = tls.Dial("tcp", addr, tlsConfig)
	} else {
		conn, err = net.Dial("tcp", addr)
	}

	if err != nil {
		return nil, err
	}

	return &TCPTransport{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

// NewTCPServerTransport creates a new TCP transport for server
func NewTCPServerTransport(conn net.Conn) *TCPTransport {
	return &TCPTransport{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
}

// ReadMessage reads a message from the TCP connection.
// Messages are framed with a 4-byte big-endian length prefix.
func (t *TCPTransport) ReadMessage() ([]byte, error) {
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

// WriteMessage writes a message to the TCP connection.
// Messages are framed with a 4-byte big-endian length prefix.
func (t *TCPTransport) WriteMessage(data []byte) error {
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

// Close closes the TCP connection
func (t *TCPTransport) Close() error {
	return t.conn.Close()
}