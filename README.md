# Aura - DNS Tunneling for WhatsApp Text Messages

**Aura** is a DNS-based tunneling system that proxies WhatsApp text messages through public DNS infrastructure. It routes traffic through standard DNS queries (AAAA records) to bypass network restrictions.

## ⚠️ Important Limitation: TEXT-ONLY
Aura **only supports WhatsApp text messages** (port 5222). All media, voice calls, and CDN traffic (port 443) are blocked by design.

## Architecture

```
Android App (SOCKS5) → Public DNS (1.1.1.1) → Aura Server (Australia) → WhatsApp (port 5222 only)
```

## Key Features

- **DNS-Based**: All traffic tunneled through AAAA DNS queries
- **No Direct Connection**: Routes through public DNS resolvers (1.1.1.1)
- **Port 5222 Only**: Enforces WhatsApp text protocol, blocks media
- **Android Compatible**: Built with gomobile for .aar library
- **Session Management**: 60-second timeout, automatic cleanup
- **Cache Busting**: Random nonces prevent DNS caching

## Quick Start

### Server Deployment (VPS in Australia)

```bash
# Prerequisites: Go 1.21+, domain with NS records pointing to your server

# Clone repository
git clone https://github.com/ArashDoDo2/Aura
cd Aura

# Build server
go build -o aura-server ./cmd/server

# Run (requires root for port 53)
sudo ./aura-server -addr :53 -zone aura.net
```

### Client Testing (Termux on Android)

```bash
# Install dependencies
pkg install golang git

# Clone and build
git clone https://github.com/ArashDoDo2/Aura
cd Aura
go build -o aura-client ./cmd/client

# Run client (starts SOCKS5 proxy on 127.0.0.1:1080)
./aura-client -dns 1.1.1.1:53 -domain aura.net
```

### Android App Integration

```bash
# Build Android library (.aar)
gomobile bind -target=android/arm64,android/amd64 -o aura.aar ./internal

# Add aura.aar to your Android project (app/libs/)
# Implement VpnService (see COMPLETE-ARCHITECTURE.md for details)
```

## Protocol Overview

### DNS Query Structure
```
[Nonce]-[Seq]-[SessionID].[Base32Data].aura.net.
```

**Example:**
```
a3f1-0001-b2c4.mzxw6ytboi.aura.net.
```

- **Nonce**: 4-char hex for cache busting
- **Seq**: 4-char hex sequence number
- **SessionID**: 4-char hex session identifier
- **Base32Data**: Encoded payload (30-byte chunks)

### Data Flow

1. **Upstream**: TCP data → 30-byte chunks → Base32 encode → DNS AAAA query
2. **Server**: Extract data → Forward to WhatsApp (port 5222) → Acknowledge
3. **Downstream**: Client polls (seq=ffff) → Server packs data into IPv6 addresses → Client extracts 16 bytes per record

## File Structure

```
Aura/
├── cmd/
│   ├── client/main.go              # Client entry point
│   └── server/main.go              # Server entry point
├── internal/
│   ├── server.go                   # DNS server + WhatsApp connection
│   ├── client.go                   # SOCKS5 proxy + DNS tunnel
│   ├── mobile.go                   # gomobile exports
│   ├── dnsutil.go                  # Encoding utilities
│   └── protocol.go                 # Query builder
├── COMPLETE-ARCHITECTURE.md        # Full technical documentation
├── ANDROID-BUILD.md                # Android build guide
├── PROJECT-GO.md                   # Go project details
└── README.md                       # This file
```

## Documentation

- **[COMPLETE-ARCHITECTURE.md](COMPLETE-ARCHITECTURE.md)** - Full system architecture, protocol details, Android integration
- **[ANDROID-BUILD.md](ANDROID-BUILD.md)** - Step-by-step Android library build
- **[PROJECT-GO.md](PROJECT-GO.md)** - Go module structure and development

## Requirements

### Server
- Go 1.21+
- Root access (for DNS port 53)
- Domain with NS record delegation
- Public IP address

### Client (Termux Testing)
- Android with Termux
- Go 1.21+
- Network access

### Android App
- gomobile installed
- Android SDK/NDK configured
- VpnService implementation (for full integration)

## Port 5222 Enforcement

Aura **hardcodes WhatsApp port 5222** in the server to ensure text-only traffic:

```go
// Server only connects to port 5222
conn, err := net.DialTimeout("tcp", "e1.whatsapp.net:5222", 5*time.Second)
```

This blocks:
- Media uploads/downloads (port 443)
- Voice/video calls (port 443)
- Status updates (port 443)
- CDN content (port 443)

## Limitations

- **Text messages only** - No media, voice, or CDN
- **High latency** - DNS queries add 500ms+ delay
- **Low throughput** - ~100 queries/sec max
- **Public DNS dependency** - Subject to rate limits
- **No E2EE** - Traffic visible to DNS resolver (WhatsApp's E2EE still protects message content)

## Use Cases

✅ **Supported:**
- Text messaging on restricted networks
- Basic WhatsApp communication
- Testing DNS tunneling concepts

❌ **Not Supported:**
- Media sharing (photos, videos, voice notes)
- Voice/video calls
- Status updates
- Group media

## Contributing

Contributions welcome! Focus areas:
- VpnService packet routing implementation
- Android UI improvements
- Protocol optimizations
- Documentation improvements

## License

MIT License - See LICENSE file

## Security Notice

- DNS queries are **visible to ISPs and DNS resolvers**
- Run your own server for privacy
- WhatsApp's end-to-end encryption remains intact
- Consider DNS-over-HTTPS (DoH) for additional privacy

## Author

**ArashDoDo2**  
GitHub: [@ArashDoDo2](https://github.com/ArashDoDo2)

---

**Warning**: This is an experimental project for educational purposes. Use at your own risk. May violate WhatsApp Terms of Service.
