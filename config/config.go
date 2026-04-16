package config

import (
	"crypto/tls"
	"time"
)

// Logger interface for logging errors and info
type Logger interface {
	Printf(format string, args ...interface{})
	Print(v ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// TransportType defines the underlying transport protocol
type TransportType int

const (
    TransportUnknown TransportType = iota // 0
    TransportWebSocket                    // 1
    TransportTCP                          // 2
    TransportTLS                          // 3
    TransportUDP                          // 4
    TransportHTTP2                        // 5
)

type ObfuscationConfig struct {
	Enabled      bool
	MaxPadding   int
	MinDelay     time.Duration
	MaxDelay     time.Duration
	PingInterval time.Duration
}

type RateLimitConfig struct {
	Enabled             bool
	StreamOpenRate      float64 // requests per second for opening streams
	FrameSendRate       float64 // requests per second for sending frames
	BurstSize           int     // burst size for rate limiters
}

type ClientConfig struct {
	ServerAddr string

	AuthToken []byte

	TLSConfig *tls.Config

	UseUTLS bool

	Fingerprint string

	// Transport specifies which transport protocol to use (required, must not be 0)
	Transport TransportType

	HandshakeTimeout time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration

	Obfuscation ObfuscationConfig

	RateLimit RateLimitConfig

	// Logger for logging errors and events (optional, defaults to log.Default())
	Logger Logger

	// OnError callback for handling errors (optional)
	OnError func(error)
}

type ServerConfig struct {
	ListenAddr string

	AllowedTokens [][]byte

	CertFile string
	KeyFile  string

	ProxyTarget string

	// Transport specifies which transport protocol to use (required, must not be 0)
	Transport TransportType

	HandshakeTimeout time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration

	Obfuscation ObfuscationConfig

	RateLimit RateLimitConfig

	// Logger for logging errors and events (optional, defaults to log.Default())
	Logger Logger

	// OnError callback for handling errors (optional)
	OnError func(error)
}

// Config is a general configuration for both client and server
type Config struct {
	ListenAddr    string
	TransportType string
	Token         string
}
