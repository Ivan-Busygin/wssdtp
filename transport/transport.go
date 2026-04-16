package transport

type Transport interface {
	ReadMessage() ([]byte, error)

	WriteMessage([]byte) error

	Close() error
}
