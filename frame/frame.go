package frame

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

const (
	TypeData  = 0x01
	TypeOpen  = 0x02
	TypeClose = 0x03
	TypePing  = 0x04
	TypePong  = 0x05
)

type FrameHeader struct {
	Type     byte
	StreamID uint16
	Length   uint16
}

func EncodeHeader(typ byte, streamID uint16, length uint16) []byte {
	buf := make([]byte, 5)
	buf[0] = typ
	binary.BigEndian.PutUint16(buf[1:3], streamID)
	binary.BigEndian.PutUint16(buf[3:5], length)
	return buf
}

func DecodeHeader(data []byte) (FrameHeader, error) {
	if len(data) < 5 {
		return FrameHeader{}, fmt.Errorf("frame header too short: got %d bytes", len(data))
	}
	return FrameHeader{
		Type:     data[0],
		StreamID: binary.BigEndian.Uint16(data[1:3]),
		Length:   binary.BigEndian.Uint16(data[3:5]),
	}, nil
}

func BuildPayload(plaintext []byte, maxPadding int) []byte {
	realLen := len(plaintext)
	if realLen > 65535 {
		realLen = 65535
		plaintext = plaintext[:65535]
	}
	paddingLen := 0
	if maxPadding > 0 {
		paddingBytes := make([]byte, 2)
		if _, err := rand.Read(paddingBytes); err == nil {
			paddingLen = int(binary.BigEndian.Uint16(paddingBytes)) % (maxPadding + 1)
		}
	}
	totalLen := 2 + realLen + paddingLen
	payload := make([]byte, totalLen)
	binary.BigEndian.PutUint16(payload[0:2], uint16(realLen))
	copy(payload[2:2+realLen], plaintext)
	if paddingLen > 0 {
		if _, err := rand.Read(payload[2+realLen:]); err != nil {
			for i := range payload[2+realLen:] {
				payload[2+realLen+i] = 0
			}
		}
	}
	return payload
}

func ExtractPayload(decrypted []byte) ([]byte, error) {
	if len(decrypted) < 2 {
		return nil, fmt.Errorf("payload too short: got %d bytes", len(decrypted))
	}
	realLen := binary.BigEndian.Uint16(decrypted[0:2])
	if int(realLen) > len(decrypted)-2 {
		return nil, fmt.Errorf("real length %d exceeds payload size %d", realLen, len(decrypted)-2)
	}
	return decrypted[2 : 2+realLen], nil
}

func EncodeFrame(typ byte, streamID uint16, plainPayload []byte, aead cipher.AEAD) ([]byte, error) {
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plainPayload, nil)
	length := uint16(len(ciphertext))
	header := EncodeHeader(typ, streamID, length)
	frame := make([]byte, 5+aead.NonceSize()+len(ciphertext))
	copy(frame[0:5], header)
	copy(frame[5:5+aead.NonceSize()], nonce)
	copy(frame[5+aead.NonceSize():], ciphertext)
	return frame, nil
}

func DecodeFrame(frame []byte, aead cipher.AEAD) (byte, uint16, []byte, error) {
	if len(frame) < 5+aead.NonceSize() {
		return 0, 0, nil, fmt.Errorf("frame too short: got %d bytes, need at least %d", len(frame), 5+aead.NonceSize())
	}

	header, err := DecodeHeader(frame[:5])
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to decode header: %w", err)
	}

	nonce := frame[5 : 5+aead.NonceSize()]
	ciphertext := frame[5+aead.NonceSize():]
	if len(ciphertext) != int(header.Length) {
		return 0, 0, nil, fmt.Errorf("ciphertext length mismatch: expected %d, got %d", header.Length, len(ciphertext))
	}

	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to decrypt frame: %w", err)
	}
	return header.Type, header.StreamID, plain, nil
}
