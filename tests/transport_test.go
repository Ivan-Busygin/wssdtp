package transport_test

import (
	"net"
	"testing"

	"wssdtp/config"
	"wssdtp/transport"
)

func TestTCPTransport(t *testing.T) {
	// Create connected pipes for testing
	serverConn, clientConn := net.Pipe()

	// Create server transport
	serverCfg := &config.ServerConfig{
		Transport: config.TransportTCP,
	}
	serverTransport, err := transport.NewServerTransport(serverCfg, serverConn)
	if err != nil {
		t.Fatalf("Failed to create server transport: %v", err)
	}

	testMessage := []byte("Hello, TCP Transport!")

	// Test write from server to client
	err = serverTransport.WriteMessage(testMessage)
	if err != nil {
		t.Fatalf("Server write failed: %v", err)
	}

	// Test read on client side (using pipe)
	buf := make([]byte, len(testMessage)+4) // +4 for length prefix
	n, err := clientConn.Read(buf)
	if err != nil {
		t.Fatalf("Client read failed: %v", err)
	}
	if n != len(testMessage)+4 {
		t.Fatalf("Expected %d bytes, got %d", len(testMessage)+4, n)
	}

	// Test server read (echo back)
	_, err = clientConn.Write(buf)
	if err != nil {
		t.Fatalf("Client write back failed: %v", err)
	}

	readMsg, err := serverTransport.ReadMessage()
	if err != nil {
		t.Fatalf("Server read failed: %v", err)
	}
	if string(readMsg) != string(testMessage) {
		t.Fatalf("Expected %s, got %s", testMessage, readMsg)
	}

	serverTransport.Close()
	clientConn.Close()
}

func TestTCPTransportLargeMessage(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	serverTransport := transport.NewTCPServerTransport(serverConn)

	// Create a large message (64KB)
	largeMessage := make([]byte, 64*1024)
	for i := range largeMessage {
		largeMessage[i] = byte(i % 256)
	}

	// Write large message
	err := serverTransport.WriteMessage(largeMessage)
	if err != nil {
		t.Fatalf("Large message write failed: %v", err)
	}

	// Read from client side
	expectedLen := len(largeMessage) + 4
	buf := make([]byte, expectedLen)
	n, err := clientConn.Read(buf)
	if err != nil {
		t.Fatalf("Large message read failed: %v", err)
	}
	if n != expectedLen {
		t.Fatalf("Expected %d bytes, got %d", expectedLen, n)
	}

	// Echo back
	_, err = clientConn.Write(buf)
	if err != nil {
		t.Fatalf("Echo write failed: %v", err)
	}

	// Read on server
	received, err := serverTransport.ReadMessage()
	if err != nil {
		t.Fatalf("Large message server read failed: %v", err)
	}

	if len(received) != len(largeMessage) {
		t.Fatalf("Large message length mismatch: expected %d, got %d", len(largeMessage), len(received))
	}
	for i, b := range received {
		if b != byte(i%256) {
			t.Fatalf("Large message content mismatch at byte %d", i)
		}
	}

	serverTransport.Close()
	clientConn.Close()
}