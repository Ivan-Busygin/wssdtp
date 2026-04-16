# WSSDTP – Secure Data Transmission Protocol

WSSDTP is a transport-level protocol that provides traffic encryption (ChaCha20-Poly1305), stream multiplexing, and DPI bypass (traffic masking). The protocol uses X25519 for key exchange and HKDF for session key derivation.

**Status**: Early development (alpha)

## Features

- **Cryptography**: X25519 + HKDF + ChaCha20-Poly1305.
- **Multiplexing**: Support for multiple independent streams over a single connection.
- **Traffic Masking**: Random padding and delays, sending PING frames to mask traffic.
- **Proxy Masking**: Server can proxy unauthenticated connections to a public WebSocket server.
- **TLS Fingerprint Spoofing**: Client can use `utls` library to emulate browser fingerprints.
- **Multiple Transports**: Support for WebSocket, TCP, TLS, UDP, HTTP/2.
- **Ease of Use**: Streams implement the `io.ReadWriteCloser` interface.
- **Extensible Protocol**: 2-byte version field supports protocol evolution.

## Project Structure

```
wssdtp/
├── transport/      # Transport abstraction and implementations (WebSocket, TCP, TLS, UDP, HTTP/2)
├── crypto/         # Cryptographic functions (X25519, HKDF, AEAD)
├── frame/          # Frame format, encoding/decoding
├── handshake/      # Handshake (82-byte message with 2-byte version)
├── session/        # Session (multiplexing, streams)
├── config/         # Client and server configurations
├── tests/          # Comprehensive test suite
├── go.mod
├── go.sum
├── LICENSE
├── README.md
├── CHANGELOG.md
├── CONTRIBUTING.md
├── PROTOCOL.md     # Complete protocol specification
└── .gitignore
```

## Requirements

- Go 1.21 or higher.
- Dependencies:
  - `github.com/gorilla/websocket`
  - `github.com/refraction-networking/utls`
  - `golang.org/x/crypto`

## Installation

Install dependencies:

```bash
go get -u github.com/gorilla/websocket
go get -u github.com/refraction-networking/utls
go get -u golang.org/x/crypto
go mod tidy
```

## Transports

WSSDTP supports several base transport protocols:

### WebSocket
- **Protocol**: RFC 6455 WebSocket
- **Features**: Full-duplex, message framing, browser compatibility
- **Use**: Web applications, proxy bypass
- **Config**: `Transport: &TransportWebSocket`

### TCP
- **Protocol**: Plain TCP with length prefix
- **Features**: Low overhead, reliable delivery
- **Use**: Direct connections, high performance
- **Config**: `Transport: &TransportTCP`

### TLS
- **Protocol**: Plain TLS with length prefix
- **Features**: Encrypted transport, certificate validation
- **Use**: Secure direct connections
- **Config**: `Transport: &TransportTLS`

### UDP
- **Protocol**: UDP with length prefix
- **Features**: Low latency, connectionless
- **Use**: Real-time applications, gaming
- **Config**: `Transport: &TransportUDP`

### HTTP/2
- **Protocol**: HTTP/2 streams with length prefix
- **Features**: Stream multiplexing, header compression, server push
- **Use**: Corporate networks, CDN integration
- **Config**: `Transport: &TransportHTTP2`

## Documentation

- **[PROTOCOL.md](PROTOCOL.md)**: Complete protocol specification for implementing WSSDTP in other languages
- **[CHANGELOG.md](CHANGELOG.md)**: Version history and changes
- **[CONTRIBUTING.md](CONTRIBUTING.md)**: Contribution guidelines

## Testing

Run the comprehensive test suite:

```bash
go test ./tests/... -v
```

Tests cover:
- Handshake protocol and message serialization
- Cryptographic operations (X25519, HKDF, ChaCha20-Poly1305)
- Frame encoding/decoding and payload handling
- Version encoding/decoding utilities

---

# WSSDTP – Протокол Безопасной Передачи Данных

WSSDTP – это протокол транспортного уровня, который обеспечивает шифрование трафика (ChaCha20‑Poly1305), мультиплексирование потоков и обход DPI (маскировка трафика). Протокол использует X25519 для обмена ключами и HKDF для вывода ключей сессии.

**Статус**: Ранняя разработка (альфа)

## Особенности

- **Криптография**: X25519 + HKDF + ChaCha20‑Poly1305.
- **Мультиплексирование**: поддержка нескольких независимых потоков через одно соединение.
- **Маскировка трафика**: случайные отступы и задержки, отправка PING фреймов для маскировки трафика.
- **Маскировка прокси**: сервер может проксировать неаутентифицированные соединения на публичный WebSocket сервер.
- **Подделка отпечатков TLS**: клиент может использовать библиотеку `utls` для эмуляции отпечатков браузеров.
- **Множественные транспорты**: поддержка WebSocket, TCP, TLS, UDP, HTTP/2.
- **Простота использования**: потоки реализуют интерфейс `io.ReadWriteCloser`.
- **Расширяемый протокол**: 2-байтовое поле версии поддерживает эволюцию протокола.

## Структура проекта

```
wssdtp/
├── transport/      # Абстракция транспорта и реализации (WebSocket, TCP, TLS, UDP, HTTP/2)
├── crypto/         # Криптографические функции (X25519, HKDF, AEAD)
├── frame/          # Формат фреймов, кодирование/декодирование
├── handshake/      # Рукопожатие (82-байтовое сообщение с 2-байтовой версией)
├── session/        # Сессия (мультиплексирование, потоки)
├── config/         # Конфигурации клиента и сервера
├── tests/          # Комплексный набор тестов
├── go.mod
├── go.sum
├── LICENSE
├── README.md
├── CHANGELOG.md
├── CONTRIBUTING.md
├── PROTOCOL.md     # Полная спецификация протокола
└── .gitignore
```

## Требования

- Go 1.21 или выше.
- Зависимости:
  - `github.com/gorilla/websocket`
  - `github.com/refraction-networking/utls`
  - `golang.org/x/crypto`

## Установка

Установите зависимости:

```bash
go get -u github.com/gorilla/websocket
go get -u github.com/refraction-networking/utls
go get -u golang.org/x/crypto
go mod tidy
```

## Транспорты

WSSDTP поддерживает несколько базовых транспортных протоколов:

### WebSocket
- **Протокол**: RFC 6455 WebSocket
- **Особенности**: Полнодуплексный, фрейминг сообщений, совместимость с браузерами
- **Применение**: Веб-приложения, обход прокси
- **Конфиг**: `Transport: &TransportWebSocket`

### TCP
- **Протокол**: Чистый TCP с префиксом длины
- **Особенности**: Низкие накладные расходы, надежная доставка
- **Применение**: Прямые соединения, высокая производительность
- **Конфиг**: `Transport: &TransportTCP`

### TLS
- **Протокол**: Чистый TLS с префиксом длины
- **Особенности**: Шифрованный транспорт, валидация сертификатов
- **Применение**: Безопасные прямые соединения
- **Конфиг**: `Transport: &TransportTLS`

### UDP
- **Протокол**: UDP с префиксом длины
- **Особенности**: Низкая задержка, без соединений
- **Применение**: Приложения реального времени, игры
- **Конфиг**: `Transport: &TransportUDP`

### HTTP/2
- **Протокол**: Потоки HTTP/2 с префиксом длины
- **Особенности**: Мультиплексирование, сжатие заголовков, server push
- **Применение**: Корпоративные сети, интеграция с CDN
- **Конфиг**: `Transport: &TransportHTTP2`

## Документация

- **[PROTOCOL.md](PROTOCOL.md)**: Полная спецификация протокола для реализации WSSDTP на других языках
- **[CHANGELOG.md](CHANGELOG.md)**: История версий и изменений
- **[CONTRIBUTING.md](CONTRIBUTING.md)**: Руководство по внесению вклада

## Тестирование

Запустите комплексный набор тестов:

```bash
go test ./tests/... -v
```

Тесты покрывают:
- Протокол рукопожатия и сериализацию сообщений
- Криптографические операции (X25519, HKDF, ChaCha20-Poly1305)
- Кодирование/декодирование фреймов и обработку полезной нагрузки
- Утилиты кодирования/декодирования версий
