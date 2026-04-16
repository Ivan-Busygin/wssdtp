package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"wssdtp/config"
	"wssdtp/handshake"
	"wssdtp/session"
	"wssdtp/transport"
)

func main() {
	// Command-line flags for configuration
	listenAddr := flag.String("listen", ":8080", "Address to listen on (host:port)")
	transportType := flag.String("transport", "websocket", "Transport type: websocket, tcp, tls, udp, http2")
	token := flag.String("token", "", "Authentication token for clients")
	flag.Parse()

	if *token == "" {
		log.Fatal("Token is required for authentication")
	}

	// Create server configuration
	var transportEnum config.TransportType
	switch *transportType {
	case "websocket":
		transportEnum = config.TransportWebSocket
	case "tcp":
		transportEnum = config.TransportTCP
	case "tls":
		transportEnum = config.TransportTLS
	case "udp":
		transportEnum = config.TransportUDP
	case "http2":
		transportEnum = config.TransportHTTP2
	default:
		log.Fatalf("Unsupported transport type: %s", *transportType)
	}

	cfg := &config.ServerConfig{
		ListenAddr: *listenAddr,
		Transport:  transportEnum,
		AllowedTokens: [][]byte{[]byte(*token)},
	}

	fmt.Printf("VPN Server listening on %s using %s transport\n", *listenAddr, *transportType)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	switch *transportType {
	case "websocket":
		startWebSocketServer(cfg)
	case "tcp":
		startTCPListener(cfg)
	case "tls":
		startTLSListener(cfg)
	case "udp":
		startUDPListener(cfg)
	case "http2":
		startHTTP2Listener(cfg)
	default:
		log.Fatalf("Unsupported transport type: %s", *transportType)
	}

	<-sigChan
	fmt.Println("Shutting down server...")
}

func startWebSocketServer(cfg *config.ServerConfig) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins for simplicity
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}
		handleConnection(conn, cfg)
	})

	log.Fatal(http.ListenAndServe(cfg.ListenAddr, nil))
}

func startTCPListener(cfg *config.ServerConfig) {
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on TCP: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept TCP connection: %v", err)
			continue
		}
		go handleConnection(conn, cfg)
	}
}

func startTLSListener(cfg *config.ServerConfig) {
	// For simplicity, assume self-signed cert or provide cert files
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load TLS cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener, err := tls.Listen("tcp", cfg.ListenAddr, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to listen on TLS: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept TLS connection: %v", err)
			continue
		}
		go handleConnection(conn, cfg)
	}
}

func startUDPListener(cfg *config.ServerConfig) {
	// UDP handling is more complex, requires custom logic for connections
	log.Fatal("UDP transport not implemented yet")
}

func startHTTP2Listener(cfg *config.ServerConfig) {
	// HTTP/2 over TLS
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load TLS cert: %v", err)
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	server := &http.Server{
		Addr:      cfg.ListenAddr,
		TLSConfig: tlsConfig,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For HTTP/2, we need to handle the connection differently
			// This is a placeholder
			log.Printf("HTTP/2 connection from %s", r.RemoteAddr)
		}),
	}

	log.Fatal(server.ListenAndServeTLS("", ""))
}

func handleConnection(conn interface{}, cfg *config.ServerConfig) {
	var rawConn net.Conn
	var tr transport.Transport
	var err error

	switch c := conn.(type) {
	case *websocket.Conn:
		// Для WebSocket создаем транспорт через helper-функцию
		// Поскольку upgrader уже использован, нам нужно создать транспорт напрямую
		tr = &webSocketConnTransport{conn: c}
		rawConn = &netConnWrapper{wsConn: c}
	case net.Conn:
		rawConn = c
		tr = &simpleNetTransport{conn: c}
	default:
		log.Printf("Unsupported connection type")
		return
	}

	defer func() {
		if rawConn != nil {
			rawConn.Close()
		}
		if tr != nil {
			tr.Close()
		}
	}()

	// Perform handshake
	sessionKey, err := handshake.PerformServerHandshake(tr, cfg.AllowedTokens, nil)
	if err != nil {
		log.Printf("Handshake failed: %v", err)
		return
	}

	fmt.Printf("Client connected from %s\n", rawConn.RemoteAddr())

	// Create session using NewSession directly
	sess, err := session.NewSession(tr, sessionKey, config.ObfuscationConfig{}, config.RateLimitConfig{}, nil, func(err error) {
		log.Printf("Session error: %v", err)
	})
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		return
	}
	defer sess.Close()

	// Start session handling (traffic forwarding)
	// Session is already running readLoop internally
	select {}
}

// simpleNetTransport wraps net.Conn to implement transport.Transport
type simpleNetTransport struct {
	conn net.Conn
}

func (t *simpleNetTransport) ReadMessage() ([]byte, error) {
	// Simple length-prefixed read for raw TCP
	var lenBuf [4]byte
	if _, err := t.conn.Read(lenBuf[:]); err != nil {
		return nil, err
	}
	msgLen := int(lenBuf[0])<<24 | int(lenBuf[1])<<16 | int(lenBuf[2])<<8 | int(lenBuf[3])
	msg := make([]byte, msgLen)
	_, err := t.conn.Read(msg)
	return msg, err
}

func (t *simpleNetTransport) WriteMessage(data []byte) error {
	// Simple length-prefixed write for raw TCP
	lenBuf := make([]byte, 4)
	lenBuf[0] = byte(len(data) >> 24)
	lenBuf[1] = byte(len(data) >> 16)
	lenBuf[2] = byte(len(data) >> 8)
	lenBuf[3] = byte(len(data))
	if _, err := t.conn.Write(lenBuf); err != nil {
		return err
	}
	_, err := t.conn.Write(data)
	return err
}

func (t *simpleNetTransport) Close() error {
	return t.conn.Close()
}

// webSocketConnTransport wraps websocket.Conn to implement transport.Transport
type webSocketConnTransport struct {
	conn *websocket.Conn
}

func (t *webSocketConnTransport) ReadMessage() ([]byte, error) {
	_, data, err := t.conn.ReadMessage()
	return data, err
}

func (t *webSocketConnTransport) WriteMessage(data []byte) error {
	return t.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (t *webSocketConnTransport) Close() error {
	return t.conn.Close()
}

// netConnWrapper wraps websocket.Conn to provide net.Conn-like interface for logging
type netConnWrapper struct {
	wsConn *websocket.Conn
}

func (w *netConnWrapper) Read(b []byte) (n int, err error) { return 0, nil }
func (w *netConnWrapper) Write(b []byte) (n int, err error) { return 0, nil }
func (w *netConnWrapper) Close() error                       { return w.wsConn.Close() }
func (w *netConnWrapper) LocalAddr() net.Addr                { return &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 0} }
func (w *netConnWrapper) RemoteAddr() net.Addr               { return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0} }
func (w *netConnWrapper) SetDeadline(t time.Time) error      { return nil }
func (w *netConnWrapper) SetReadDeadline(t time.Time) error  { return nil }
func (w *netConnWrapper) SetWriteDeadline(t time.Time) error { return nil }