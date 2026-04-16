# WSSDTP VPN Usage Guide

## Introduction

WSSDTP is a simple VPN protocol designed to bypass DPI (Deep Packet Inspection) and provide secure traffic tunneling. This guide explains how to use the example applications to create a VPN server and client that fully redirect device traffic.

## Requirements

- Go 1.19 or higher
- Root privileges (for TUN interface setup)
- Linux/macOS/Windows (with TUN support)

## Compilation

Compile the examples:

```bash
go build ./examples/simple_vpn_server.go
go build ./examples/simple_vpn_client.go
```

This will create executable files `simple_vpn_server` and `simple_vpn_client`.

## Running the Server

Run the server on a machine with a public IP (or use NAT/port forwarding):

```bash
sudo ./simple_vpn_server -listen=:8080 -transport=websocket -token=your-secret-token
```

Parameters:
- `-listen`: Listening address (default `:8080`)
- `-transport`: Transport type (websocket, tcp, tls, udp, http2)
- `-token`: Secret token for client authentication

The server will listen for incoming connections and be ready to accept clients.

## Running the Client

On the client machine, run the client:

```bash
sudo ./simple_vpn_client -server=server-ip:8080 -transport=websocket -token=your-secret-token
```

Parameters:
- `-server`: Server address (IP:port)
- `-transport`: Transport type (must match server)
- `-token`: Authentication token (must match server)

## Network Configuration

After starting the client and server, you need to configure the network interfaces for traffic redirection.

### On the Client

1. Find the TUN interface IP address (usually `tun0`):
   ```bash
   ip addr show tun0
   ```

2. Configure IP address and route:
   ```bash
   sudo ip addr add 10.0.0.2/24 dev tun0
   sudo ip route add default via 10.0.0.1 dev tun0 metric 1
   ```

3. Disable reverse route for the server (to avoid loops):
   ```bash
   sudo ip route del server-ip via original-gateway
   ```

### On the Server

1. Configure the TUN interface:
   ```bash
   sudo ip addr add 10.0.0.1/24 dev tun0
   sudo sysctl -w net.ipv4.ip_forward=1
   sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
   ```

   Replace `eth0` with your external interface.

## Testing

1. Check connectivity:
   ```bash
   ping 8.8.8.8
   ```

2. Check IP address:
   ```bash
   curl ifconfig.me
   ```
   Should show the server's IP.

3. Check speed:
   ```bash
   speedtest-cli
   ```

---

# Руководство по использованию WSSDTP VPN

## Введение

WSSDTP - это простой VPN-протокол, разработанный для обхода DPI (Deep Packet Inspection) и обеспечения безопасного туннелирования трафика. Этот гайд объясняет, как использовать примеры приложений для создания VPN-сервера и клиента, которые полностью перенаправляют трафик устройства.

## Требования

- Go 1.19 или выше
- Root-права (для настройки TUN-интерфейсов)
- Linux/macOS/Windows (с поддержкой TUN)

## Компиляция

Скомпилируйте примеры:

```bash
go build ./examples/simple_vpn/simple_vpn_server.go
go build ./examples/simple_vpn/simple_vpn_client.go
```

Это создаст исполняемые файлы `simple_vpn_server` и `simple_vpn_client`.

## Запуск сервера

Запустите сервер на машине с публичным IP (или используйте NAT/port forwarding):

```bash
sudo ./simple_vpn_server -listen=:8080 -transport=websocket -token=your-secret-token
```

Параметры:
- `-listen`: Адрес для прослушивания (по умолчанию `:8080`)
- `-transport`: Тип транспорта (websocket, tcp, tls, udp, http2)
- `-token`: Секретный токен для аутентификации клиентов

Сервер будет слушать входящие соединения и готов принимать клиентов.

## Запуск клиента

На клиентской машине запустите клиент:

```bash
sudo ./simple_vpn_client -server=server-ip:8080 -transport=websocket -token=your-secret-token
```

Параметры:
- `-server`: Адрес сервера (IP:порт)
- `-transport`: Тип транспорта (должен совпадать с сервером)
- `-token`: Токен аутентификации (должен совпадать с сервером)

## Настройка сети

После запуска клиента и сервера необходимо настроить сетевые интерфейсы для перенаправления трафика.

### На клиенте

1. Найдите IP-адрес TUN-интерфейса (обычно `tun0`):
   ```bash
   ip addr show tun0
   ```

2. Настройте IP-адрес и маршрут:
   ```bash
   sudo ip addr add 10.0.0.2/24 dev tun0
   sudo ip route add default via 10.0.0.1 dev tun0 metric 1
   ```

3. Отключите обратный маршрут для сервера (чтобы избежать петель):
   ```bash
   sudo ip route del server-ip via original-gateway
   ```

### На сервере

1. Настройте TUN-интерфейс:
   ```bash
   sudo ip addr add 10.0.0.1/24 dev tun0
   sudo sysctl -w net.ipv4.ip_forward=1
   sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
   ```

   Замените `eth0` на ваш внешний интерфейс.

## Проверка работы

1. Проверьте подключение:
   ```bash
   ping 8.8.8.8
   ```

2. Проверьте IP-адрес:
   ```bash
   curl ifconfig.me
   ```
   Должен показывать IP сервера.

3. Проверьте скорость:
   ```bash
   speedtest-cli
   ```

## Как работает обход DPI

Протокол WSSDTP использует несколько техник для обхода DPI:

1. **Маскировка под обычный трафик**: Использует WebSocket, HTTP/2 или TLS для маскировки VPN-трафика под обычные веб-запросы.

2. **Шифрование**: ChaCha20-Poly1305 шифрует все данные, делая их нечитаемыми для DPI.

3. **Обфускация**: Транспортные слои могут имитировать легитимный трафик (например, WebSocket под видом чата).

4. **Фрагментация**: Пакеты разбиваются и переупорядочиваются для усложнения анализа.

5. **Адаптивность**: Протокол может переключаться между транспортами в зависимости от условий сети.

## Troubleshooting

### Проблемы с подключением

- Проверьте, что токен совпадает на клиенте и сервере
- Убедитесь, что порт открыт и доступен
- Проверьте логи сервера на ошибки handshake

### Проблемы с трафиком

- Проверьте настройки iptables на сервере
- Убедитесь, что TUN-интерфейсы настроены правильно
- Проверьте маршруты: `ip route show`

### Низкая производительность

- Попробуйте другой транспорт (UDP обычно быстрее)
- Проверьте CPU и память на сервере
- Уменьшите MTU если есть проблемы с фрагментацией

### DPI блокирует

- Попробуйте другой транспорт (HTTP/2 или TLS)
- Используйте obfuscation если доступно
- Проверьте на другом порту (443 для HTTPS)

## Безопасность

- Используйте сильные токены
- Запускайте сервер за NAT/firewall
- Регулярно обновляйте Go и зависимости
- Мониторьте логи на подозрительную активность

## Расширенная конфигурация

Для продакшена рассмотрите:
- Использование TLS с валидными сертификатами
- Настройку rate limiting
- Добавление логирования и мониторинга
- Использование systemd для автозапуска