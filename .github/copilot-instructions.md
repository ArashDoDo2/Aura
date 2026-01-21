# Aura: DNS Tunneling System - AI Agent Guide

## Architecture Overview

**Core Concept**: DNS-only covert channel for WhatsApp text messaging through public DNS infrastructure. All data flows through AAAA record queries (IPv6 addresses).

```
Android App (SOCKS5) → Public DNS → Authoritative Server → WhatsApp (port 5222 only)
```

**Critical Design Decision**: Port 5222 enforcement = TEXT-ONLY. Blocks media/CDN (port 443) intentionally. This is architectural, not configurable.

**Dual Android Integration Paths**:
1. **gomobile (MethodChannel)**: `internal/mobile.go` → `.aar` → Kotlin VpnService (legacy, has NDK compatibility issues)
2. **FFI (Direct)**: `bridge.go` → `libaura.so` → Dart FFI (current, recommended - see [BUILD-FFI-ANDROID.md](../BUILD-FFI-ANDROID.md))

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
- **Sequence-aware buffering**: `pendingChunks` map orders out-of-order packets
- **Background reader goroutine**: `startReader()` continuously reads from target connection
- **TLS handshake optimization**: Buffers complete ClientHello for single-write transmission

### Client ([internal/client.go](../internal/client.go))
- SOCKS5 proxy on localhost:1080
- **System DNS support**: Empty `DNSServer` uses system resolver (see `getEffectiveDNSServer()`)
- **Dual-mode operation**: Auto-detects SOCKS5 (0x05) vs Transparent TCP
- **TLS fast-path**: Buffers complete TLS ClientHello for single-write transmission
- Sequential polling at 100ms intervals
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
- **FFI Bridge**: `lib/aura_bridge.dart` → `libaura.so` (direct CGo calls)
- System DNS auto-detection in UI (leave DNS field empty)
- No VpnService required for FFI approach

**Alternative**: [AuraMobileClient/](../AuraMobileClient/) uses gomobile `.aar` + Kotlin VpnService (legacy)

## Build & Deployment

**Standard Go build** (all platforms):
```bash
# Server (requires root on Unix for port 53)
go build -o aura-server ./cmd/server
sudo ./aura-server -domain yourdomain.com. -addr :53

# Client (standalone, Termux-compatible)
go build -o aura-client ./cmd/client
./aura-client -dns "" -domain yourdomain.com.  # System DNS
```

**Android Integration (Choose ONE approach)**:

**Option 1: FFI (Recommended - No NDK issues)**:
```powershell
# Prerequisites: Set NDK env vars (see BUILD-FFI-ANDROID.md)
$env:ANDROID_NDK_HOME = "$env:LOCALAPPDATA\Android\Sdk\ndk\29.0.14206865"
$env:CC = "$env:ANDROID_NDK_HOME\toolchains\llvm\prebuilt\windows-x86_64\bin\aarch64-linux-android21-clang.cmd"
$env:CGO_ENABLED = "1"; $env:GOOS = "android"; $env:GOARCH = "arm64"

# Build .so (bridge.go exports StartAura/StopAura via CGo)
go build -buildmode=c-shared -o libaura.so ./bridge.go

# Deploy to Flutter
Copy-Item libaura.so flutter_aura\android\app\src\main\jniLibs\arm64-v8a\
cd flutter_aura
flutter build apk --debug
```

**Option 2: gomobile (Legacy - NDK compatibility issues)**:
```bash
# Note: May fail with "unsupported Android platform version" on NDK 26+
gomobile bind -target=android/arm64,android/amd64 -o aura.aar ./internal
# Copy aura.aar to AuraMobileClient/app/libs/
```

**Key Differences**:
- **FFI**: Direct CGo → Dart FFI, simpler, no MethodChannel, fewer layers
- **gomobile**: JNI → Kotlin → MethodChannel → Dart, more complex, NDK version-sensitive

**Configuration**: Flags OR environment variables
- Server: `AURA_DOMAIN`, `AURA_LISTEN_ADDR`
- Client: `AURA_DNS_SERVER` (empty = system), `AURA_DOMAIN`, `AURA_SOCKS5_PORT`

## Code Conventions

1. **Base32 encoding**: Always lowercase, no padding (`b32Encoder.WithPadding(base32.NoPadding)`)
2. **Error handling**: Log and return; server sends DNS error codes (RcodeFormatError, RcodeServerFailure)
3. **Concurrency**: Session map protected by `sync.Mutex`, per-session locks for conn operations
4. **Background goroutines**: Server uses `startReader()` goroutine per session for continuous downstream reads
5. **Sequence ordering**: Server `processPendingChunks()` ensures in-order delivery via `expectedSeq` tracking
6. **DNS library**: github.com/miekg/dns for all DNS operations
7. **Module name**: `github.com/ArashDoDo2/Aura` (import paths must match)
8. **Transparent TCP mode**: `handleSocks5Conn` uses `Peek(1)` to detect protocol - if NOT 0x05, immediately divert to `handleTunnel` with `return` to prevent SOCKS5 handshake execution
9. **TLS optimization**: Both client and server buffer complete TLS ClientHello for single-write transmission (avoids TCP fragmentation)

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

**FFI Architecture** (Recommended):
```
Flutter (Dart) → dart:ffi → libaura.so (CGo) → Go Client
```

**Key files**:
- [bridge.go](../bridge.go): CGo exports `StartAura`/`StopAura` for FFI
- [flutter_aura/lib/aura_bridge.dart](../flutter_aura/lib/aura_bridge.dart): Dart FFI bindings
- [flutter_aura/lib/main.dart](../flutter_aura/lib/main.dart): UI with DNS/domain inputs

**gomobile Architecture** (Legacy):
```
Flutter (Dart) ↔ MethodChannel ↔ Kotlin ↔ JNI ↔ Go
```

**Key files**:
- [internal/mobile.go](../internal/mobile.go): gomobile exports for `.aar`
- [AuraMobileClient/](../AuraMobileClient/): Complete Android app with VpnService

**MethodChannel protocol** (gomobile only):
```dart
startVpn({dnsServer: String, domain: String}) -> "" or error
stopVpn() -> "" or error
getStatus() -> "running" | "stopped"
```

## Documentation Structure

- [COMPLETE-ARCHITECTURE.md](../COMPLETE-ARCHITECTURE.md): Deep dive into protocol and Android VPN setup
- [FLUTTER-BUILD.md](../FLUTTER-BUILD.md): Complete Flutter + Go build guide
- [BUILD-FFI-ANDROID.md](../BUILD-FFI-ANDROID.md): FFI approach (recommended, no NDK issues)
- [ANDROID-BUILD.md](../ANDROID-BUILD.md): Pure Android (no Flutter) build instructions
- [SYSTEM-DNS-IMPLEMENTATION.md](../SYSTEM-DNS-IMPLEMENTATION.md): System resolver integration details
- [PROJECT-GO.md](../PROJECT-GO.md): Persian documentation (راهنمای فارسی)

## Troubleshooting

### gomobile build fails
```powershell
# NDK version conflicts - use FFI approach instead (BUILD-FFI-ANDROID.md)
# Or downgrade NDK: Android Studio → SDK Manager → NDK 25.2.9519653
```

### Flutter FFI build fails
```powershell
# Check NDK paths
$env:ANDROID_NDK_HOME = "$env:LOCALAPPDATA\Android\Sdk\ndk\29.0.14206865"
Get-Item $env:ANDROID_NDK_HOME  # Should exist

# Verify .so file
Get-Item libaura.so | Select-Object Name, Length  # Should be ~7 MB

# Clean rebuild
flutter clean
flutter pub get
flutter build apk --debug
```

### Client "domain must end with dot" error
```bash
# CORRECT: example.com.
# WRONG: example.com
./aura-client -domain example.com.  # Always include trailing dot
```

### Server not responding to queries
```powershell
# Check DNS server running
sudo netstat -ulnp | grep :53

# Test direct query
dig @YOUR_SERVER_IP test.yourdomain.com. AAAA

# Check server logs for "does not end with domain" errors
```
