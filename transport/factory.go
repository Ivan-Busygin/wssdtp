package transport

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"wssdtp/config"
)

// NewClientTransport creates a transport based on the configuration
func NewClientTransport(cfg *config.ClientConfig) (Transport, error) {
	if cfg.Transport == 0 {
		return nil, errors.New("transport type must be specified in client config")
	}
	switch cfg.Transport {
	case config.TransportWebSocket:
		return NewWebSocketClientTransport(cfg.ServerAddr, cfg.TLSConfig, cfg.UseUTLS, cfg.Fingerprint)
	case config.TransportTCP:
		return NewTCPClientTransport(cfg.ServerAddr, cfg.TLSConfig)
	case config.TransportTLS:
		if cfg.TLSConfig == nil {
			return nil, errors.New("TLS config required for TLS transport")
		}
		return NewTLSClientTransport(cfg.ServerAddr, cfg.TLSConfig)
	case config.TransportUDP:
		// For UDP, we need to parse the address to separate local and remote
		// For now, assume "local:remote" format or just use remote as both
		return NewUDPClientTransport("0.0.0.0:0", cfg.ServerAddr)
	case config.TransportHTTP2:
		return NewHTTP2ClientTransport(cfg.ServerAddr, cfg.TLSConfig)
	default:
		return nil, errors.New("unsupported transport type")
	}
}

// NewServerTransport creates a server transport based on the configuration and connection
func NewServerTransport(cfg *config.ServerConfig, conn interface{}) (Transport, error) {
	if cfg.Transport == 0 {
		return nil, errors.New("transport type must be specified in server config")
	}
	switch cfg.Transport {
	case config.TransportWebSocket:
		// For WebSocket server, we need http.ResponseWriter and *http.Request
		if wsConn, ok := conn.(*WebSocketConnection); ok {
			return NewWebSocketServerTransport(wsConn.Writer, wsConn.Request, wsConn.Upgrader)
		}
		return nil, errors.New("WebSocket server transport requires WebSocketConnection")
	case config.TransportTCP:
		if tcpConn, ok := conn.(net.Conn); ok {
			return NewTCPServerTransport(tcpConn), nil
		}
		return nil, errors.New("TCP server transport requires net.Conn")
	case config.TransportTLS:
		if tlsConn, ok := conn.(*tls.Conn); ok {
			return NewTLSServerTransport(tlsConn), nil
		}
		return nil, errors.New("TLS server transport requires *tls.Conn")
	case config.TransportUDP:
		if udpConn, ok := conn.(*UDPConnection); ok {
			return NewUDPServerTransport(udpConn.Conn, udpConn.RemoteAddr), nil
		}
		return nil, errors.New("UDP server transport requires UDPConnection")
	case config.TransportHTTP2:
		if httpConn, ok := conn.(io.ReadWriteCloser); ok {
			return NewHTTP2ServerTransport(httpConn)
		}
		return nil, errors.New("HTTP/2 server transport requires io.ReadWriteCloser")
	default:
		return nil, errors.New("unsupported transport type")
	}
}

// WebSocketConnection holds WebSocket server connection data
type WebSocketConnection struct {
	Writer   http.ResponseWriter
	Request  *http.Request
	Upgrader websocket.Upgrader
}

// UDPConnection holds UDP server connection data
type UDPConnection struct {
	Conn       *net.UDPConn
	RemoteAddr *net.UDPAddr
}

// CreateListener creates a listener based on the configuration
func CreateListener(cfg *config.Config) (interface{}, error) {
	switch cfg.TransportType {
	case "websocket":
		// For WebSocket, we need an HTTP server
		return nil, errors.New("WebSocket listener not implemented yet")
	case "tcp":
		return net.Listen("tcp", cfg.ListenAddr)
	case "tls":
		// For TLS, need certs, but for simplicity
		return nil, errors.New("TLS listener not implemented yet")
	case "udp":
		return net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 0})
	case "http2":
		return nil, errors.New("HTTP/2 listener not implemented yet")
	default:
		return nil, errors.New("unsupported transport type")
	}
}