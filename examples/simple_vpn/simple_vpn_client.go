package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"wssdtp/config"
	"wssdtp/session"
	"github.com/songgao/water"
)

func main() {
	// Command line flags for easy configuration
	var serverAddr = flag.String("server", "localhost:8080", "Server address to connect to")
	var transportType = flag.String("transport", "websocket", "Transport type: websocket, tcp, tls, udp, http2")
	var authToken = flag.String("token", "", "Authentication token for server")
	flag.Parse()

	// Create TUN interface for capturing and injecting IP packets
	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		log.Fatal("Failed to create TUN interface:", err)
	}
	log.Printf("TUN interface created: %s", ifce.Name())

	// Note: To fully redirect all device traffic, you need to configure the TUN interface with IP and routes.
	// This requires root privileges. Example commands (run as root):
	// ip addr add 10.0.0.1/24 dev <ifce.Name()>
	// ip link set <ifce.Name()> up
	// ip route add default dev <ifce.Name()>  # This redirects all traffic through the TUN
	// For this example, manual configuration is assumed.

	// Parse transport type from string
	var tr config.TransportType
	switch *transportType {
	case "websocket":
		tr = config.TransportWebSocket
	case "tcp":
		tr = config.TransportTCP
	case "tls":
		tr = config.TransportTLS
	case "udp":
		tr = config.TransportUDP
	case "http2":
		tr = config.TransportHTTP2
	default:
		log.Fatal("Unknown transport type:", *transportType)
	}

	// Create client configuration
	cfg := &config.ClientConfig{
		ServerAddr: *serverAddr,
		Transport:  tr,
		AuthToken:  []byte(*authToken),
		Logger:     nil, // Use default logger
	}

	// Establish session with server
	sess, err := session.NewClientSession(cfg, cfg.AuthToken)
	if err != nil {
		log.Fatal("Failed to create session:", err)
	}
	defer sess.Close()

	// Open a stream for traffic data
	stream, err := sess.OpenStream()
	if err != nil {
		log.Fatal("Failed to open stream:", err)
	}
	defer stream.Close()

	// Goroutine to read packets from TUN and send to server via stream
	go func() {
		buf := make([]byte, 1500) // Standard MTU size
		for {
			n, err := ifce.Read(buf)
			if err != nil {
				log.Printf("TUN read error: %v", err)
				return
			}
			_, err = stream.Write(buf[:n])
			if err != nil {
				log.Printf("Stream write error: %v", err)
				return
			}
		}
	}()

	// Read packets from server via stream and write to TUN
	buf := make([]byte, 1500)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			log.Printf("Stream read error: %v", err)
			return
		}
		_, err = ifce.Write(buf[:n])
		if err != nil {
			log.Printf("TUN write error: %v", err)
			return
		}
	}

	// Wait for interrupt signal to exit gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("Shutting down VPN client")
}