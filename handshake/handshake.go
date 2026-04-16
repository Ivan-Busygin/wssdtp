// Package handshake implements the WSSDTP protocol handshake mechanism.
// The handshake establishes a secure connection by exchanging cryptographic keys
// and authenticating both client and server using tokens.
package handshake

import (
	"errors"
	"fmt"

	"wssdtp/config"
	"wssdtp/crypto"
	"wssdtp/transport"
)

type defaultLogger struct{}

func (d *defaultLogger) Printf(format string, args ...interface{}) {}
func (d *defaultLogger) Print(v ...interface{})                    {}
func (d *defaultLogger) Error(msg string, args ...interface{})    {}
func (d *defaultLogger) Debug(msg string, args ...interface{})    {}
func (d *defaultLogger) Info(msg string, args ...interface{})     {}
func (d *defaultLogger) Warn(msg string, args ...interface{})     {}

// PerformClientHandshake initiates the handshake process from the client side.
// It generates a key pair, creates a handshake message with the client's public key,
// authentication token, and random data, then sends it to the server.
// After receiving the server's response, it validates the version and token,
// computes the shared secret, and derives the session key.
//
// Parameters:
//   - conn: A Transport connection to the server
//   - authToken: A 16-byte authentication token shared with the server
//
// Returns:
//   - sessionKey: A 32-byte key for encrypting subsequent communication
//   - error: Any error that occurred during the handshake
//
// The handshake message format is 82 bytes:
//   - Version (2 bytes): Protocol version as uint16
//   - Random (32 bytes): Client-generated random data
//   - PublicKey (32 bytes): Client's X25519 public key
//   - AuthToken (16 bytes): Authentication token
func PerformClientHandshake(conn transport.Transport, authToken []byte, logger config.Logger) ([]byte, error) {
	if logger == nil {
		logger = &defaultLogger{}
	}
	logger.Info("starting client handshake")
	if len(authToken) != 16 {
		return nil, errors.New("auth token must be 16 bytes")
	}

	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	msg := HandshakeMessage{
		Version: EncodeVersion(ProtocolVersionMajor, ProtocolVersionMinor),
	}
	copy(msg.PublicKey[:], pub)
	copy(msg.AuthToken[:], authToken)
	if err := msg.GenerateRandom(); err != nil {
		return nil, fmt.Errorf("failed to generate random data: %w", err)
	}

	if err := conn.WriteMessage(msg.Marshal()); err != nil {
		return nil, fmt.Errorf("failed to send handshake message: %w", err)
	}

	respData, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to receive handshake response: %w", err)
	}
	if len(respData) != 82 {
		return nil, fmt.Errorf("invalid handshake response length: expected 82, got %d", len(respData))
	}
	var resp HandshakeMessage
	if err := resp.Unmarshal(respData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal handshake response: %w", err)
	}

	respMajor, respMinor := DecodeVersion(resp.Version)
	if respMajor != ProtocolVersionMajor || respMinor != ProtocolVersionMinor {
		return nil, fmt.Errorf("unsupported protocol version: server is %d.%d, client is %s",
			respMajor, respMinor, ProtocolVersionString)
	}

	if string(resp.AuthToken[:]) != string(authToken) {
		logger.Error("auth token mismatch")
		return nil, errors.New("auth token mismatch")
	}

	shared, err := crypto.ComputeSharedSecret(priv, resp.PublicKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	salt := make([]byte, 64)
	copy(salt[:32], msg.Random[:])
	copy(salt[32:], resp.Random[:])

	info := []byte("wssdtp-session-key")
	sessionKey, err := crypto.HKDFExpand(shared, salt, info, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive session key: %w", err)
	}

	logger.Info("handshake successful, session key derived")
	return sessionKey, nil
}

// PerformServerHandshake handles the handshake process from the server side.
// It receives the client's handshake message, validates the protocol version
// and authentication token against the list of allowed tokens, generates its own
// key pair, and responds with a handshake message containing the server's public key.
// Finally, it computes the shared secret and derives the session key.
//
// Parameters:
//   - conn: A Transport connection to the client
//   - allowedTokens: A slice of 16-byte authentication tokens that are accepted
//
// Returns:
//   - sessionKey: A 32-byte key for encrypting subsequent communication
//   - error: Any error that occurred during the handshake
//
// The server validates that the client's token matches one of the allowed tokens
// and that the protocol version is compatible. If validation fails, the connection
// should be closed by the caller.
func PerformServerHandshake(conn transport.Transport, allowedTokens [][]byte, logger config.Logger) ([]byte, error) {
	if logger == nil {
		logger = &defaultLogger{}
	}
	logger.Info("starting server handshake")

	clientData, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to receive handshake message: %w", err)
	}
	if len(clientData) != 82 {
		return nil, fmt.Errorf("invalid handshake message length: expected 82, got %d", len(clientData))
	}
	var clientMsg HandshakeMessage
	if err := clientMsg.Unmarshal(clientData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal handshake message: %w", err)
	}

	clientMajor, clientMinor := DecodeVersion(clientMsg.Version)
	if clientMajor != ProtocolVersionMajor || clientMinor != ProtocolVersionMinor {
		return nil, fmt.Errorf("unsupported protocol version: client is %d.%d, server is %s",
			clientMajor, clientMinor, ProtocolVersionString)
	}

	tokenValid := false
	for _, token := range allowedTokens {
		if len(token) == 16 && string(clientMsg.AuthToken[:]) == string(token) {
			tokenValid = true
			break
		}
	}
	if !tokenValid {
		logger.Error("invalid auth token")
		return nil, errors.New("invalid auth token")
	}

	priv, pub, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	resp := HandshakeMessage{
		Version: EncodeVersion(ProtocolVersionMajor, ProtocolVersionMinor),
	}
	copy(resp.PublicKey[:], pub)
	copy(resp.AuthToken[:], clientMsg.AuthToken[:]) // Echo back the token
	if err := resp.GenerateRandom(); err != nil {
		return nil, fmt.Errorf("failed to generate random data: %w", err)
	}

	if err := conn.WriteMessage(resp.Marshal()); err != nil {
		return nil, fmt.Errorf("failed to send handshake response: %w", err)
	}

	shared, err := crypto.ComputeSharedSecret(priv, clientMsg.PublicKey[:])
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	salt := make([]byte, 64)
	copy(salt[:32], clientMsg.Random[:])
	copy(salt[32:], resp.Random[:])

	info := []byte("wssdtp-session-key")
	sessionKey, err := crypto.HKDFExpand(shared, salt, info, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive session key: %w", err)
	}

	logger.Info("handshake successful, session key derived")
	return sessionKey, nil
}
