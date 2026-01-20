# Aura: DNS Tunneling System - AI Agent Guide

## Architecture Overview

**Core Concept**: DNS-only covert channel for WhatsApp text messaging through public DNS infrastructure. All data flows through AAAA record queries (IPv6 addresses).

```
Android App (SOCKS5) → Public DNS → Authoritative Server → WhatsApp (port 5222 only)
```

**Critical Design Decision**: Port 5222 enforcement = TEXT-ONLY. Blocks media/CDN (port 443) intentionally. This is architectural, not configurable.

## Protocol Structure

DNS queries follow strict naming convention:
```
[Nonce(4hex)]-[Seq(4hex)]-[SessionID(4hex)].[Base32Data].<domain>.
Example: a3f1-0001-b2c4.mzxw6ytboi.example.com.
```

- **Nonce**: Cache-busting (prevents DNS resolver caching)
- **Seq**: Packet ordering (0000-FFFF, ffff = polling request)
- **SessionID**: Session identifier (4-char hex)
- **Base32Data**: Payload (lowercase, no padding)

See [internal/server.go](../internal/server.go#L12-L19) for `ParseQueryName` implementation.

## Data Flow Patterns

**Upstream** (Client → Server):
1. Fragment TCP data into 30-byte chunks ([internal/client.go](../internal/client.go#L23))
2. Base32 encode (lowercase, no padding)
3. Send as AAAA query
4. Server decodes and forwards to WhatsApp:5222

**Downstream** (Server → Client):
1. Client polls with seq=`ffff`
2. Server packs data into IPv6 addresses (16 bytes each)
3. Multiple AAAA records if needed
4. Client extracts bytes from each IPv6, concatenates

## Key Components

### Server ([internal/server.go](../internal/server.go))
- Authoritative DNS responder
- Session management with 60s timeout
- **Hardcoded**: `WhatsAppHost = "e1.whatsapp.net"` and `WhatsAppPort = 5222`
- Rejects queries with "media" or "cdn" substrings

### Client ([internal/client.go](../internal/client.go))
- SOCKS5 proxy on localhost:1080
- **System DNS support**: Empty `DNSServer` uses system resolver (see `getEffectiveDNSServer()`)
- Sequential polling at 500ms intervals
- Session ID generated per client instance

### Mobile Bridge ([internal/mobile.go](../internal/mobile.go))
- gomobile-compatible exports for Android
- `StartTunnel(dnsServer, domain)` - Main entry point (returns error string)
- `StopTunnel()` - Graceful shutdown
- `IsRunning()` - Status check
- `GetStatus()` - Detailed status string
- Global client singleton pattern
- Legacy `StartAuraClient()` for backward compatibility

### Flutter Integration ([flutter_aura/](../flutter_aura/))
- Material Design UI with VPN controls
- MethodChannel bridge: `com.aura.proxy/vpn`
- Kotlin VpnService wraps Go engine
- System DNS auto-detection in UI
- Optional per-app VPN for WhatsApp only

## Build & Deployment

**Standard Go build**:
```bash
go build -o aura-server ./cmd/server
go build -o aura-client ./cmd/client
```

**Android .aar**:
```bash
gomobile bind -target=android/arm64,android/amd64 -o aura.aar ./internal
```

**Flutter app**:
```bash
cd flutter_aura
flutter pub get
flutter run  # Requires aura.aar in android/app/libs/
```

**Configuration**: Flags OR environment variables
- Server: `AURA_DOMAIN`, `AURA_LISTEN_ADDR`
- Client: `AURA_DNS_SERVER` (empty = system), `AURA_DOMAIN`, `AURA_SOCKS5_PORT`

## Code Conventions

1. **Base32 encoding**: Always lowercase, no padding (`b32Encoder.WithPadding(base32.NoPadding)`)
2. **Error handling**: Log and return; server sends DNS error codes (RcodeFormatError, RcodeServerFailure)
3. **Concurrency**: Session map protected by `sync.Mutex`, per-session locks for conn operations
4. **Non-blocking I/O**: `SetReadDeadline(10ms)` for downstream reads ([internal/server.go](../internal/server.go#L139))
5. **DNS library**: github.com/miekg/dns for all DNS operations

## Testing Workflow

**Quick test on Android (Termux)**:
```bash
pkg install golang git
go run ./cmd/client -dns YOUR_SERVER_IP:53 -domain example.com.
```

**Local testing**:
1. Run server: `sudo ./aura-server -domain test.local.`
2. Run client: `./aura-client -dns 127.0.0.1:53 -domain test.local.`
3. Configure app to use SOCKS5 proxy at 127.0.0.1:1080

## Critical Constraints

- **Port 5222 only**: No media support by design, not a bug
- **DNS label limits**: 63 chars max per label, 30-byte upstream chunks
- **IPv6 payload**: 16 bytes per AAAA record
- **Session timeout**: 60 seconds of inactivity triggers cleanup
- **Cache busting**: Required for real-time operation

## Common Gotchas

1. **Domain must end with dot**: `example.com.` not `example.com`
2. **Server requires root**: Port 53 needs elevated privileges on Unix
3. **System DNS on mobile**: Empty DNS string uses device resolver - preferred for Android
4. **gomobile compatibility**: Only use gomobile-compatible types in exported functions (no error return on StartAuraClient)
5. **DNS verification**: Server checks `HasSuffix(q.Name, s.Domain)` before processing

## Flutter + Go Bridge

**Architecture**: Flutter (Dart) ↔ MethodChannel ↔ Kotlin ↔ JNI ↔ Go

**Key files**:
- [flutter_aura/lib/vpn_manager.dart](../flutter_aura/lib/vpn_manager.dart): MethodChannel client
- [AuraVpnService.kt](../flutter_aura/android/app/src/main/kotlin/com/aura/flutter_aura/AuraVpnService.kt): VpnService + Go bridge
- [MainActivity.kt](../flutter_aura/android/app/src/main/kotlin/com/aura/flutter_aura/MainActivity.kt): MethodChannel handler

**MethodChannel protocol**:
```dart
startVpn({dnsServer: String, domain: String}) -> "" or error
stopVpn() -> "" or error
getStatus() -> "running" | "stopped"
```

**VPN packet forwarding**: Basic implementation in `AuraVpnService.forwardPackets()` - TODO: full SOCKS5 integration

## Documentation Structure

- [COMPLETE-ARCHITECTURE.md](../COMPLETE-ARCHITECTURE.md): Deep dive into protocol and Android VPN setup
- [FLUTTER-BUILD.md](../FLUTTER-BUILD.md): Complete Flutter + Go build guide
- [ANDROID-BUILD.md](../ANDROID-BUILD.md): Pure Android (no Flutter) build instructions
- [SYSTEM-DNS-IMPLEMENTATION.md](../SYSTEM-DNS-IMPLEMENTATION.md): System resolver integration details
- [PROJECT-GO.md](../PROJECT-GO.md): Persian documentation (راهنمای فارسی)
