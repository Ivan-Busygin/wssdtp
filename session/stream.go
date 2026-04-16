package session

import (
	"errors"
	"io"
	"sync"

	"wssdtp/frame"
)

type Stream struct {
	id      uint16
	session *Session
	readCh  chan []byte
	closeCh chan struct{}
	closed  bool
	mu      sync.Mutex
}

func newStream(id uint16, s *Session) *Stream {
	return &Stream{
		id:      id,
		session: s,
		readCh:  make(chan []byte, 10),
		closeCh: make(chan struct{}),
	}
}

func (st *Stream) Read(p []byte) (int, error) {
	select {
	case data, ok := <-st.readCh:
		if !ok {
			return 0, io.EOF
		}
		n := copy(p, data)
		return n, nil
	case <-st.closeCh:
		return 0, io.EOF
	}
}

func (st *Stream) Write(p []byte) (int, error) {
	if st.isClosed() {
		return 0, errors.New("stream closed")
	}
	err := st.session.sendFrame(frame.TypeData, st.id, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (st *Stream) Close() error {
	if st.isClosed() {
		return nil
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.closed {
		return nil
	}
	st.closed = true
	close(st.closeCh)
	_ = st.session.sendFrame(frame.TypeClose, st.id, nil)
	return nil
}

func (st *Stream) pushData(data []byte) {
	clone := make([]byte, len(data))
	copy(clone, data)
	select {
	case st.readCh <- clone:
	case <-st.closeCh:
	}
}

func (st *Stream) closeStream() {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.closed {
		return
	}
	st.closed = true
	close(st.closeCh)
}

func (st *Stream) isClosed() bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.closed
}
