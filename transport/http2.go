package transport

import (
	"crypto/tls"
	"errors"
	"io"
)

// HTTP2Transport implements Transport interface over HTTP/2
// Note: This is a placeholder implementation. Full HTTP/2 support requires
// integration with HTTP servers and clients.
type HTTP2Transport struct {
	// Placeholder - not implemented yet
}

// NewHTTP2ClientTransport creates a new HTTP/2 transport for client
func NewHTTP2ClientTransport(url string, tlsConfig *tls.Config) (*HTTP2Transport, error) {
	return nil, errors.New("HTTP/2 transport not implemented yet")
}

// NewHTTP2ServerTransport creates a new HTTP/2 transport for server
func NewHTTP2ServerTransport(body io.ReadWriteCloser) (Transport, error) {
	return nil, errors.New("HTTP/2 transport not implemented yet")
}

// ReadMessage reads a message from the HTTP/2 connection
func (t *HTTP2Transport) ReadMessage() ([]byte, error) {
	return nil, errors.New("HTTP/2 transport not implemented yet")
}

// WriteMessage writes a message to the HTTP/2 connection
func (t *HTTP2Transport) WriteMessage(data []byte) error {
	return errors.New("HTTP/2 transport not implemented yet")
}

// Close closes the HTTP/2 transport
func (t *HTTP2Transport) Close() error {
	return errors.New("HTTP/2 transport not implemented yet")
}