# WSSDTP Protocol Specification

## Overview

WSSDTP is a transport-level protocol that provides end-to-end encryption, stream multiplexing, and traffic masking to bypass DPI (Deep Packet Inspection).

**Version**: 0.0 (Prerelease)  
**Transport**: Multiple (WebSocket, TCP, TLS, UDP, HTTP/2)  
**Encryption**: ChaCha20-Poly1305 AEAD  
**Key Exchange**: X25519 ECDH  
**Key Derivation**: HKDF-SHA256  

## Protocol Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │    │   WSSDTP │    │   WebSocket     │
│                 │    │   - Handshake   │    │   Transport     │
│   Streams       │◄──►│   - Encryption  │◄──►│                 │
│   (io.Reader/   │    │   - Multiplex   │    │   TCP/TLS       │
│    io.Writer)   │    │   - Obfuscation │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 1. Transport Layer

WSSDTP is transport-agnostic and can operate over several base protocols. The transport layer provides a simple message-based interface:

- `ReadMessage()` → `([]byte, error)` - Read the next message
- `WriteMessage([]byte)` → `error` - Write a message
- `Close()` → `error` - Close the connection

### Supported Transports

#### WebSocket Transport
- **Protocol**: RFC 6455 WebSocket
- **Message Framing**: WebSocket binary messages
- **Connection**: HTTP upgrade handshake
- **Features**: Browser compatibility, proxy bypass

#### TCP Transport
- **Protocol**: Plain TCP
- **Message Framing**: 4-byte length prefix big-endian + data
- **Connection**: Direct TCP connection
- **Features**: Low overhead, high performance

#### TLS Transport
- **Protocol**: Plain TLS over TCP
- **Message Framing**: 4-byte length prefix big-endian + data
- **Connection**: TLS handshake
- **Features**: Built-in encryption, certificate validation

#### UDP Transport
- **Protocol**: UDP
- **Message Framing**: 2-byte length prefix big-endian + data (max 65535 bytes)
- **Connection**: Connectionless UDP
- **Features**: Low latency, stateless

#### HTTP/2 Transport
- **Protocol**: HTTP/2 streams
- **Message Framing**: 4-byte length prefix big-endian + data
- **Connection**: HTTP/2 handshake
- **Features**: Stream multiplexing, header compression

## 2. Handshake Protocol

The handshake establishes cryptographic keys and authenticates peers.

### Handshake Message Format

All handshake messages are exactly 82 bytes:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          Version (uint16)     |                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               +
|                                                               |
|                    Random (32 bytes)                          |
|                                                               |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|                                                               |
|                                                               |
|                    Public Key (32 bytes)                      |
|                                                               |
|                                                               |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|                    Auth Token (16 bytes)                      |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**Fields**:
- **Version**: Protocol version (major.minor encoded as uint16, big-endian)
  - High byte: Major version
  - Low byte: Minor version
  - Example: Version 1.2 = 0x0102
- **Random**: 32 bytes of cryptographically secure random data
- **Public Key**: X25519 public key (32 bytes)
- **Auth Token**: 16-byte authentication token (shared secret)

### Handshake Sequence

```
Client                          Server
  |                               |
  |  HandshakeMessage{Version,    |
  |    Random_C, PublicKey_C,     |
  |    AuthToken}                 |
  |------------------------------>|
  |                               |
  |  HandshakeMessage{Version,    |
  |    Random_S, PublicKey_S,     |
  |    AuthToken}                 |
  |<------------------------------|
  |                               |
```

### Key Derivation

1. **Shared Secret**: ECDH using X25519
   ```
   shared_secret = X25519(client_private_key, server_public_key)
                 = X25519(server_private_key, client_public_key)
   ```

2. **Session Key**: HKDF derivation
   ```
   salt = client_random + server_random  (64 bytes)
   info = "wssdtp-session-key"
   session_key = HKDF-Expand(shared_secret, salt, info, 32)
   ```

## 3. Frame Protocol

After handshake, all communication uses encrypted frames.

### Frame Header (5 bytes)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Type |                   Stream ID (uint16)                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   Length (uint16)                           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**Fields**:
- **Type** (1 byte): Frame type
- **Stream ID** (2 bytes): Stream identifier (big-endian)
- **Length** (2 bytes): Length of encrypted payload (big-endian)

### Frame Types

| Type | Value | Description |
|------|-------|-------------|
| DATA | 0x01 | Encrypted stream data |
| OPEN | 0x02 | Open new stream |
| CLOSE| 0x03 | Close stream |
| PING | 0x04 | Keep-alive ping |
| PONG | 0x05 | Keep-alive pong |

### Encrypted Frame Format

```
+-------------------+-------------------+-------------------+
| Frame Header (5) | Nonce (12 bytes) | Ciphertext (N)    |
+-------------------+-------------------+-------------------+
```

**Encryption**:
- **Algorithm**: ChaCha20-Poly1305 AEAD
- **Key**: 32-byte session key
- **Nonce**: 12 bytes (random per frame)
- **Plaintext**: Frame payload (may include padding)

### DATA Frame Payload Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          Real Length (uint16) |                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               +
|                                                               |
|                    Data (variable)                            |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Padding (variable)                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**Fields**:
- **Real Length**: Actual data length (big-endian uint16)
- **Data**: Application data
- **Padding**: Random bytes for traffic masking

## 4. Stream Multiplexing

### Stream Lifecycle

1. **OPEN**: Client sends OPEN frame to create stream
2. **DATA**: Bidirectional encrypted data transfer
3. **CLOSE**: Either peer sends CLOSE frame to terminate stream

### Stream IDs

- **Range**: 1-65535
- **Direction**: Bidirectional (same ID for both directions)
- **Uniqueness**: IDs unique within session

## 5. Traffic Obfuscation

### Padding

- Random padding added to DATA frames
- Configurable maximum padding size
- Padding bytes are cryptographically random

### Timing Obfuscation

- Configurable delays between frames
- Random delay within min/max range
- PING frames sent to mask traffic

### Connection Masking

- Server can proxy unauthenticated connections
- Client can spoof TLS fingerprints

## 6. Error Handling

### Protocol Errors

- **Version Mismatch**: Incompatible protocol versions
- **Authentication Failure**: Invalid authentication token
- **Decryption Failure**: Invalid ciphertext/tag
- **Invalid Frame**: Corrupted frame data

### Recovery

- Close connection on protocol errors
- No automatic retry (application responsibility)

## 7. Security Considerations

### Cryptographic Security

- **Forward Secrecy**: ECDH provides perfect forward secrecy
- **AEAD**: ChaCha20-Poly1305 provides confidentiality and integrity
- **Key Derivation**: HKDF prevents weak key derivation

### Implementation Security

- **Random Generation**: Use cryptographically secure RNG
- **Key Management**: Zero sensitive data after use
- **Timing Attacks**: Constant-time cryptographic operations

## 8. Implementation Guide

### Required Libraries

- **Go**: golang.org/x/crypto (ChaCha20-Poly1305, X25519, HKDF)
- **WebSocket**: gorilla/websocket
- **TLS Spoofing**: refraction-networking/utls (optional)

### Key Functions

```go
// Handshake
PerformClientHandshake(conn, authToken) → sessionKey
PerformServerHandshake(conn, allowedTokens) → sessionKey

// Version handling
EncodeVersion(major, minor) → uint16
DecodeVersion(version) → major, minor

// Frame operations
EncodeFrame(type, streamID, payload, aead) → frame
DecodeFrame(frame, aead) → type, streamID, payload

// Stream operations
OpenStream() → stream
stream.Read(data)
stream.Write(data)
stream.Close()
```

## 9. Version History

- **0.0**: Initial prerelease
  - Basic handshake and encryption
  - Stream multiplexing
  - Traffic obfuscation
  - Frame-based communication

## 10. References

- [RFC 6455: The WebSocket Protocol](https://tools.ietf.org/html/rfc6455)
- [RFC 7748: Elliptic Curves for Security](https://tools.ietf.org/html/rfc7748)
- [RFC 8439: ChaCha20 and Poly1305 for IETF Protocols](https://tools.ietf.org/html/rfc8439)

---

# Спецификация протокола WSSDTP

## Обзор

WSSDTP - это протокол транспортного уровня, который обеспечивает сквозное шифрование, мультиплексирование потоков и маскировку трафика для обхода DPI (Deep Packet Inspection).

**Версия**: 0.0 (Предрелиз)  
**Транспорт**: Множественный (WebSocket, TCP, TLS, UDP, HTTP/2)  
**Шифрование**: ChaCha20-Poly1305 AEAD  
**Обмен ключами**: X25519 ECDH  
**Вывод ключей**: HKDF-SHA256  

## Архитектура протокола

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Приложение    │    │   WSSDTP │    │   WebSocket     │
│                 │    │   - Рукопожатие │    │   Транспорт     │
│   Потоки        │◄──►│   - Шифрование  │◄──►│                 │
│   (io.Reader/   │    │   - Мультиплекс │    │   TCP/TLS       │
│    io.Writer)   │    │   - Маскировка  │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 1. Транспортный уровень

WSSDTP не зависит от транспорта и может работать через несколько базовых протоколов. Транспортный уровень предоставляет простой интерфейс на основе сообщений:

- `ReadMessage()` → `([]byte, error)` - Прочитать следующее сообщение
- `WriteMessage([]byte)` → `error` - Записать сообщение
- `Close()` → `error` - Закрыть соединение

### Поддерживаемые транспорты

#### WebSocket Транспорт
- **Протокол**: RFC 6455 WebSocket
- **Фрейминг сообщений**: Бинарные сообщения WebSocket
- **Соединение**: HTTP upgrade handshake
- **Особенности**: Совместимость с браузерами, обход прокси

#### TCP Транспорт
- **Протокол**: Чистый TCP
- **Фрейминг сообщений**: 4-байтовый префикс длины big-endian + данные
- **Соединение**: Прямое TCP соединение
- **Особенности**: Низкие накладные расходы, высокая производительность

#### TLS Транспорт
- **Протокол**: Чистый TLS поверх TCP
- **Фрейминг сообщений**: 4-байтовый префикс длины big-endian + данные
- **Соединение**: TLS handshake
- **Особенности**: Встроенное шифрование, валидация сертификатов

#### UDP Транспорт
- **Протокол**: UDP
- **Фрейминг сообщений**: 2-байтовый префикс длины big-endian + данные (макс 65535 байт)
- **Соединение**: Без соединения UDP
- **Особенности**: Низкая задержка, отсутствие состояния соединения

#### HTTP/2 Транспорт
- **Протокол**: Потоки HTTP/2
- **Фрейминг сообщений**: 4-байтовый префикс длины big-endian + данные
- **Соединение**: HTTP/2 handshake
- **Особенности**: Мультиплексирование потоков, сжатие заголовков

## 2. Протокол рукопожатия

Рукопожатие устанавливает криптографические ключи и аутентифицирует пиры.

### Формат сообщения рукопожатия

Все сообщения рукопожатия точно 82 байта:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          Version (uint16)     |                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               +
|                                                               |
|                    Random (32 bytes)                          |
|                                                               |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|                                                               |
|                                                               |
|                    Public Key (32 bytes)                      |
|                                                               |
|                                                               |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|                    Auth Token (16 bytes)                      |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**Поля**:
- **Version**: Версия протокола (major.minor закодирована как uint16, big-endian)
  - Старший байт: Major версия
  - Младший байт: Minor версия
  - Пример: Версия 1.2 = 0x0102
- **Random**: 32 байта криптографически безопасных случайных данных
- **Public Key**: Публичный ключ X25519 (32 байта)
- **Auth Token**: 16-байтовый токен аутентификации (общий секрет)

### Последовательность рукопожатия

```
Клиент                          Сервер
  |                               |
  |  HandshakeMessage{Version,    |
  |    Random_C, PublicKey_C,     |
  |    AuthToken}                 |
  |------------------------------>|
  |                               |
  |  HandshakeMessage{Version,    |
  |    Random_S, PublicKey_S,     |
  |    AuthToken}                 |
  |<------------------------------|
  |                               |
```

### Вывод ключей

1. **Общий секрет**: ECDH с использованием X25519
   ```
   shared_secret = X25519(client_private_key, server_public_key)
                 = X25519(server_private_key, client_public_key)
   ```

2. **Ключ сессии**: Вывод HKDF
   ```
   salt = client_random + server_random  (64 bytes)
   info = "wssdtp-session-key"
   session_key = HKDF-Expand(shared_secret, salt, info, 32)
   ```

## 3. Протокол фреймов

После рукопожатия вся коммуникация использует зашифрованные фреймы.

### Заголовок фрейма (5 байт)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Type |                   Stream ID (uint16)                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   Length (uint16)                           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**Поля**:
- **Type** (1 байт): Тип фрейма
- **Stream ID** (2 байта): Идентификатор потока (big-endian)
- **Length** (2 байта): Длина зашифрованной полезной нагрузки (big-endian)

### Типы фреймов

| Тип | Значение | Описание |
|-----|----------|----------|
| DATA | 0x01 | Зашифрованные данные потока |
| OPEN | 0x02 | Открыть новый поток |
| CLOSE| 0x03 | Закрыть поток |
| PING | 0x04 | Keep-alive ping |
| PONG | 0x05 | Keep-alive pong |

### Формат зашифрованного фрейма

```
+-------------------+-------------------+-------------------+
| Заголовок фрейма (5) | Nonce (12 байт) | Шифртекст (N)    |
+-------------------+-------------------+-------------------+
```

**Шифрование**:
- **Алгоритм**: ChaCha20-Poly1305 AEAD
- **Ключ**: 32-байтовый ключ сессии
- **Nonce**: 12 байт (случайный для каждого фрейма)
- **Открытый текст**: Полезная нагрузка фрейма (может включать отступ)

### Формат полезной нагрузки DATA фрейма

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          Real Length (uint16) |                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               +
|                                                               |
|                    Data (variable)                            |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Padding (variable)                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**Поля**:
- **Real Length**: Фактическая длина данных (big-endian uint16)
- **Data**: Данные приложения
- **Padding**: Случайные байты для маскировки трафика

## 4. Мультиплексирование потоков

### Жизненный цикл потока

1. **OPEN**: Клиент отправляет OPEN фрейм для создания потока
2. **DATA**: Двунаправленная передача зашифрованных данных
3. **CLOSE**: Любой пир отправляет CLOSE фрейм для завершения потока

### ID потоков

- **Диапазон**: 1-65535
- **Направление**: Двунаправленный (одинаковый ID для обоих направлений)
- **Уникальность**: ID уникальны в сессии

## 5. Маскировка трафика

### Отступ

- Случайный отступ добавляется к DATA фреймам
- Настраиваемый максимальный размер отступа
- Байты отступа криптографически случайны

### Маскировка тайминга

- Настраиваемые задержки между фреймами
- Случайная задержка в пределах min/max
- PING фреймы отправляются для маскировки трафика

### Маскировка соединения

- Сервер может проксировать неаутентифицированные соединения
- Клиент может подделывать отпечатки TLS

## 6. Обработка ошибок

### Ошибки протокола

- **Несоответствие версий**: Несовместимые версии протокола
- **Сбой аутентификации**: Недействительный токен аутентификации
- **Сбой дешифрования**: Недействительный шифртекст/тег
- **Недействительный фрейм**: Повреждённые данные фрейма

### Восстановление

- Закрыть соединение при ошибках протокола
- Нет автоматического повтора (ответственность приложения)

## 7. Соображения безопасности

### Криптографическая безопасность

- **Пересылка вперёд**: ECDH обеспечивает совершенную пересылку вперёд
- **AEAD**: ChaCha20-Poly1305 обеспечивает конфиденциальность и целостность
- **Вывод ключей**: HKDF предотвращает слабый вывод ключей

### Безопасность реализации

- **Генерация случайных чисел**: Использовать криптографически безопасный RNG
- **Управление ключами**: Обнулять чувствительные данные после использования
- **Атаки по времени**: Постоянное время криптографических операций

## 8. Руководство по реализации

### Необходимые библиотеки

- **Go**: golang.org/x/crypto (ChaCha20-Poly1305, X25519, HKDF)
- **WebSocket**: gorilla/websocket
- **Подделка TLS**: refraction-networking/utls (опционально)

### Ключевые функции

```go
// Рукопожатие
PerformClientHandshake(conn, authToken) → sessionKey
PerformServerHandshake(conn, allowedTokens) → sessionKey

// Обработка версий
EncodeVersion(major, minor) → uint16
DecodeVersion(version) → major, minor

// Операции с фреймами
EncodeFrame(type, streamID, payload, aead) → frame
DecodeFrame(frame, aead) → type, streamID, payload

// Операции с потоками
OpenStream() → stream
stream.Read(data)
stream.Write(data)
stream.Close()
```

## 9. История версий

- **0.0**: Начальный предрелиз
  - Базовое рукопожатие и шифрование
  - Мультиплексирование потоков
  - Traffic obfuscation
  - Frame-based communication

## 10. References

- [RFC 6455: WebSocket Protocol](https://tools.ietf.org/html/rfc6455)
- [RFC 7748: Elliptic Curves for Security](https://tools.ietf.org/html/rfc7748)
- [RFC 8439: ChaCha20 and Poly1305](https://tools.ietf.org/html/rfc8439)
- [RFC 5869: HMAC-based Key Derivation](https://tools.ietf.org/html/rfc5869)