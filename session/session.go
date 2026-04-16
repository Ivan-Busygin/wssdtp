package session

import (
	"crypto/cipher"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"wssdtp/config"
	"wssdtp/crypto"
	"wssdtp/frame"
	"wssdtp/handshake"
	"wssdtp/transport"
)

type Session struct {
	transport transport.Transport
	aead      cipher.AEAD
	streams   map[uint16]*Stream
	nextID    uint16
	mu        sync.RWMutex
	closeOnce sync.Once
	closed    bool
	readDone  chan struct{}

	logger  config.Logger
	onError func(error)

	obfuscation struct {
		enabled      bool
		maxPadding   int
		minDelay     time.Duration
		maxDelay     time.Duration
		pingInterval time.Duration
	}
	lastSendTime time.Time
	sendMu       sync.Mutex

	rateLimit struct {
		enabled       bool
		streamLimiter *rate.Limiter
		frameLimiter  *rate.Limiter
	}
}

type defaultLogger struct {
	*log.Logger
}

func (d *defaultLogger) Error(msg string, args ...interface{}) {
	d.Printf("Error: "+msg, args...)
}

func (d *defaultLogger) Debug(msg string, args ...interface{}) {
	d.Printf("Debug: "+msg, args...)
}

func (d *defaultLogger) Info(msg string, args ...interface{}) {
	d.Printf("Info: "+msg, args...)
}

func (d *defaultLogger) Warn(msg string, args ...interface{}) {
	d.Printf("Warn: "+msg, args...)
}

func NewSession(tr transport.Transport, sessionKey []byte, obfuscation config.ObfuscationConfig, rateLimit config.RateLimitConfig, logger config.Logger, onError func(error)) (*Session, error) {
	aead, err := crypto.NewAEAD(sessionKey)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = &defaultLogger{log.Default()}
	}
	s := &Session{
		transport: tr,
		aead:      aead,
		streams:   make(map[uint16]*Stream),
		nextID:    1,
		readDone:  make(chan struct{}),
		logger:    logger,
		onError:   onError,
	}
	s.logger.Info("session created with obfuscation enabled: %v, rate limiting enabled: %v", obfuscation.Enabled, rateLimit.Enabled)
	s.obfuscation.enabled = obfuscation.Enabled
	s.obfuscation.maxPadding = obfuscation.MaxPadding
	s.obfuscation.minDelay = obfuscation.MinDelay
	s.obfuscation.maxDelay = obfuscation.MaxDelay
	s.obfuscation.pingInterval = obfuscation.PingInterval

	s.rateLimit.enabled = rateLimit.Enabled
	if s.rateLimit.enabled {
		s.rateLimit.streamLimiter = rate.NewLimiter(rate.Limit(rateLimit.StreamOpenRate), rateLimit.BurstSize)
		s.rateLimit.frameLimiter = rate.NewLimiter(rate.Limit(rateLimit.FrameSendRate), rateLimit.BurstSize)
	}
	go s.readLoop()
	if s.obfuscation.enabled && s.obfuscation.pingInterval > 0 {
		go s.pingLoop()
	}
	return s, nil
}

func (s *Session) OpenStream() (*Stream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errors.New("session closed")
	}
	if s.rateLimit.enabled && !s.rateLimit.streamLimiter.Allow() {
		s.logger.Warn("stream open rate limit exceeded")
		return nil, errors.New("rate limit exceeded for opening streams")
	}
	id := s.nextID
	s.nextID++
	stream := newStream(id, s)
	s.streams[id] = stream
	if err := s.sendFrame(frame.TypeOpen, id, nil); err != nil {
		delete(s.streams, id)
		s.logger.Error("failed to send open frame for stream %d: %v", id, err)
		return nil, err
	}
	s.logger.Info("stream %d opened", id)
	return stream, nil
}

func (s *Session) sendFrame(typ byte, streamID uint16, plainData []byte) error {
	if s.rateLimit.enabled && !s.rateLimit.frameLimiter.Allow() {
		s.logger.Warn("frame send rate limit exceeded for type %d, stream %d", typ, streamID)
		return errors.New("rate limit exceeded for sending frames")
	}
	if s.obfuscation.enabled && typ == frame.TypeData {
		plainData = frame.BuildPayload(plainData, s.obfuscation.maxPadding)
	}
	msg, err := frame.EncodeFrame(typ, streamID, plainData, s.aead)
	if err != nil {
		s.logger.Error("failed to encode frame type %d, stream %d: %v", typ, streamID, err)
		return err
	}

	if s.obfuscation.enabled && typ == frame.TypeData {
		s.sendMu.Lock()
		now := time.Now()
		if !s.lastSendTime.IsZero() {
			elapsed := now.Sub(s.lastSendTime)
			if elapsed > s.obfuscation.minDelay {
				delay := s.obfuscation.minDelay
				if s.obfuscation.maxDelay > s.obfuscation.minDelay {
					randDelay := s.obfuscation.minDelay + time.Duration(rand.Int63n(int64(s.obfuscation.maxDelay-s.obfuscation.minDelay)))
					delay = randDelay
				}
				if delay > 0 {
					time.Sleep(delay)
				}
			}
		}
		s.lastSendTime = time.Now()
		s.sendMu.Unlock()
	}
	err = s.transport.WriteMessage(msg)
	if err != nil {
		s.logger.Error("failed to write message to transport: %v", err)
	}
	s.logger.Debug("sent frame type %d, stream %d, size %d", typ, streamID, len(msg))
	return err
}

func (s *Session) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in readLoop: %v", r)
			s.logger.Printf("Error: %v", err)
			if s.onError != nil {
				s.onError(err)
			}
			s.Close()
		}
	}()
	defer close(s.readDone)
	for {
		msg, err := s.transport.ReadMessage()
		if err != nil {
			s.logger.Error("failed to read message from transport: %v", err)
			if s.onError != nil {
				s.onError(err)
			}
			s.Close()
			return
		}
		typ, streamID, plain, err := frame.DecodeFrame(msg, s.aead)
		if err != nil {
			s.logger.Error("failed to decode frame: %v", err)
			if s.onError != nil {
				s.onError(err)
			}
			s.Close()
			return
		}
		switch typ {
		case frame.TypeData:
			data, err := frame.ExtractPayload(plain)
			if err != nil {
				s.logger.Error("failed to extract payload from frame (streamID %d): %v", streamID, err)
				if s.onError != nil {
					s.onError(err)
				}
				continue
			}
			s.mu.RLock()
			stream, ok := s.streams[streamID]
			s.mu.RUnlock()
			if ok {
				stream.pushData(data)
				s.logger.Debug("received data frame for stream %d, size %d", streamID, len(data))
			} else {
				s.logger.Warn("received data frame for unknown stream %d", streamID)
			}
		case frame.TypeOpen:
			s.mu.Lock()
			if _, exists := s.streams[streamID]; !exists {
				stream := newStream(streamID, s)
				s.streams[streamID] = stream
				s.logger.Info("stream %d opened by remote", streamID)
			} else {
				s.logger.Warn("attempt to open already existing stream %d", streamID)
			}
			s.mu.Unlock()
		case frame.TypeClose:
			s.mu.Lock()
			if stream, ok := s.streams[streamID]; ok {
				stream.closeStream()
				delete(s.streams, streamID)
				s.logger.Info("stream %d closed", streamID)
			} else {
				s.logger.Warn("attempt to close unknown stream %d", streamID)
			}
			s.mu.Unlock()
		case frame.TypePing:
			if err := s.sendFrame(frame.TypePong, 0, nil); err != nil {
				s.logger.Error("failed to send pong frame: %v", err)
				if s.onError != nil {
					s.onError(err)
				}
			} else {
				s.logger.Debug("sent pong frame in response to ping")
			}
		case frame.TypePong:
			s.logger.Debug("received pong frame")
		default:
			s.logger.Warn("received unknown frame type %d", typ)
		}
	}
}

func (s *Session) pingLoop() {
	ticker := time.NewTicker(s.obfuscation.pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if s.isClosed() {
				return
			}
			if err := s.sendFrame(frame.TypePing, 0, nil); err != nil {
				s.logger.Error("failed to send ping frame: %v", err)
				if s.onError != nil {
					s.onError(err)
				}
			} else {
				s.logger.Debug("sent ping frame")
			}
		case <-s.readDone:
			return
		}
	}
}

func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		for _, stream := range s.streams {
			stream.closeStream()
		}
		s.streams = nil
		s.mu.Unlock()
		_ = s.transport.Close()
	})
	return nil
}

func (s *Session) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// NewClientSession creates a new client session with the specified configuration and authentication token.
// It establishes the transport connection, performs the handshake, and initializes the session.
func NewClientSession(cfg *config.ClientConfig, authToken []byte) (*Session, error) {
	tr, err := transport.NewClientTransport(cfg)
	if err != nil {
		return nil, err
	}

	sessionKey, err := handshake.PerformClientHandshake(tr, authToken, cfg.Logger)
	if err != nil {
		tr.Close()
		return nil, err
	}

	sess, err := NewSession(tr, sessionKey, cfg.Obfuscation, cfg.RateLimit, cfg.Logger, cfg.OnError)
	if err != nil {
		tr.Close()
		return nil, err
	}

	return sess, nil
}
