package transport

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

type WebSocketTransport struct {
	conn *websocket.Conn
}

func NewWebSocketServerTransport(w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader) (*WebSocketTransport, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &WebSocketTransport{conn: conn}, nil
}

func NewWebSocketClientTransport(addr string, tlsConfig *tls.Config, useUTLS bool, fingerprint string) (*WebSocketTransport, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	var netConn net.Conn

	if useUTLS {
		// Temporarily disabled utls due to version compatibility
		conn, err := tls.Dial("tcp", u.Host, tlsConfig)
		if err != nil {
			return nil, err
		}
		netConn = conn
		/*
		utlsConfig := &utls.Config{
			ServerName:         tlsConfig.ServerName,
			InsecureSkipVerify: tlsConfig.InsecureSkipVerify,
		}
		switch fingerprint {
		case "Chrome":
			utlsConfig.ClientHelloID = utls.HelloChrome_Auto
		case "Firefox":
			utlsConfig.ClientHelloID = utls.HelloFirefox_Auto
		case "Safari":
			utlsConfig.ClientHelloID = utls.HelloSafari_Auto
		default:
			utlsConfig.ClientHelloID = utls.HelloChrome_Auto
		}
		conn, err := utls.Dial("tcp", u.Host, utlsConfig)
		if err != nil {
			return nil, err
		}
		netConn = conn
		*/
	} else {
		conn, err := tls.Dial("tcp", u.Host, tlsConfig)
		if err != nil {
			return nil, err
		}
		netConn = conn
	}

	dialer := websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return netConn, nil
		},
		TLSClientConfig: nil,
	}

	conn, _, err := dialer.Dial(addr, nil)
	if err != nil {
		return nil, err
	}
	return &WebSocketTransport{conn: conn}, nil
}

func (t *WebSocketTransport) ReadMessage() ([]byte, error) {
	_, data, err := t.conn.ReadMessage()
	return data, err
}

func (t *WebSocketTransport) WriteMessage(data []byte) error {
	return t.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (t *WebSocketTransport) Close() error {
	return t.conn.Close()
}
