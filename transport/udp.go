package transport

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"
)

// UDPTransport implements Transport interface over UDP
type UDPTransport struct {
	conn        *net.UDPConn
	remoteAddr  *net.UDPAddr
	readTimeout time.Duration
	writeMu     sync.Mutex // Protect writes since UDP is not thread-safe for writes
}

// NewUDPClientTransport creates a new UDP transport for client
func NewUDPClientTransport(localAddr, remoteAddr string) (*UDPTransport, error) {
	localUDPAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, err
	}

	remoteUDPAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", localUDPAddr)
	if err != nil {
		return nil, err
	}

	return &UDPTransport{
		conn:        conn,
		remoteAddr:  remoteUDPAddr,
		readTimeout: 30 * time.Second, // Default timeout
	}, nil
}

// NewUDPServerTransport creates a new UDP transport for server
func NewUDPServerTransport(conn *net.UDPConn, remoteAddr *net.UDPAddr) *UDPTransport {
	return &UDPTransport{
		conn:        conn,
		remoteAddr:  remoteAddr,
		readTimeout: 30 * time.Second,
	}
}

// ReadMessage reads a message from the UDP connection.
// Messages are framed with a 2-byte big-endian length prefix.
// This is a blocking operation with timeout.
func (t *UDPTransport) ReadMessage() ([]byte, error) {
	buffer := make([]byte, 65535) // Max UDP packet size

	// Set read deadline
	if t.readTimeout > 0 {
		t.conn.SetReadDeadline(time.Now().Add(t.readTimeout))
	}

	n, addr, err := t.conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, err
	}

	// Verify sender (for server-side, we expect messages from the connected client)
	if t.remoteAddr != nil && !addr.IP.Equal(t.remoteAddr.IP) && addr.Port != t.remoteAddr.Port {
		// Ignore messages from unexpected senders
		return t.ReadMessage() // Recursive call to wait for correct sender
	}

	if n < 2 {
		return nil, errors.New("UDP packet too small for length prefix")
	}

	// Read length prefix (2 bytes for UDP efficiency)
	length := int(binary.BigEndian.Uint16(buffer[:2]))
	if length > n-2 {
		return nil, errors.New("UDP packet length mismatch")
	}

	// Extract message data
	data := make([]byte, length)
	copy(data, buffer[2:2+length])

	return data, nil
}

// WriteMessage writes a message to the UDP connection.
// Messages are framed with a 2-byte big-endian length prefix.
func (t *UDPTransport) WriteMessage(data []byte) error {
	if len(data) > 65533 { // 65535 - 2 bytes for length
		return errors.New("message too large for UDP")
	}

	// Prepare buffer with length prefix
	buffer := make([]byte, 2+len(data))
	binary.BigEndian.PutUint16(buffer[:2], uint16(len(data)))
	copy(buffer[2:], data)

	// Protect concurrent writes
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	// Set write deadline
	if t.readTimeout > 0 {
		t.conn.SetWriteDeadline(time.Now().Add(t.readTimeout))
	}

	_, err := t.conn.WriteToUDP(buffer, t.remoteAddr)
	return err
}

// Close closes the UDP connection
func (t *UDPTransport) Close() error {
	return t.conn.Close()
}

// SetTimeout sets read/write timeout for the UDP transport
func (t *UDPTransport) SetTimeout(timeout time.Duration) {
	t.readTimeout = timeout
}