# Aura Flutter Client

Flutter UI for Aura DNS Tunnel with Go engine integration.

## Features

✅ **Beautiful Material Design UI**
✅ **System DNS Auto-Detection**
✅ **Real-time VPN Status**
✅ **Easy Configuration**
✅ **WhatsApp-Only Mode** (optional)

## Quick Start

### Prerequisites
```bash
flutter --version  # 3.0+
go version         # 1.21+
gomobile version
```

### Build & Run

```powershell
# 1. Build Go engine
cd ..
gomobile bind -target=android -o aura.aar ./internal

# 2. Copy AAR
Copy-Item aura.aar flutter_aura\android\app\libs\

# 3. Build Flutter app
cd flutter_aura
flutter pub get
flutter run
```

## Usage

1. **Leave DNS empty** for automatic system DNS (recommended)
2. **Enter your domain** with trailing dot (e.g., `tunnel.example.com.`)
3. **Tap CONNECT** and grant VPN permission
4. **Use WhatsApp** - traffic routes through Aura tunnel

## Architecture

```
UI (Flutter/Dart)
    ↕ MethodChannel: com.aura.proxy/vpn
Native Bridge (Kotlin)
    ↕ JNI (gomobile)
Go Engine (Aura)
    ↕ SOCKS5 (localhost:1080)
DNS Tunnel
```

## Configuration

### Per-App VPN (WhatsApp Only)

Edit `android/app/src/main/kotlin/.../AuraVpnService.kt`:

```kotlin
// Uncomment this block:
try {
    builder.addAllowedApplication("com.whatsapp")
} catch (e: Exception) {
    Log.w(TAG, "Could not set per-app VPN: ${e.message}")
}
```

### Custom DNS

Leave empty for system DNS, or enter:
- `1.1.1.1:53` - Cloudflare
- `8.8.8.8:53` - Google
- Your server IP

## Troubleshooting

### Build Issues

```powershell
# Clean everything
flutter clean
rm -r android/app/libs/aura.aar
gomobile clean

# Rebuild
gomobile bind -target=android -o aura.aar ../internal
Copy-Item aura.aar android\app\libs\
flutter pub get
flutter build apk
```

### VPN Issues

```powershell
# Check logs
flutter logs
# Or
adb logcat -s AuraVpnService
```

**Common problems:**
- Missing AAR in `libs/` folder
- Domain without trailing dot
- VPN permission denied

## Development

### File Structure

```
lib/
├── main.dart           # UI + App logic
└── vpn_manager.dart    # MethodChannel bridge

android/app/src/main/kotlin/com/aura/flutter_aura/
├── MainActivity.kt     # MethodChannel handler
└── AuraVpnService.kt   # VPN service + Go integration
```

### MethodChannel API

**Dart → Kotlin:**
```dart
// Start VPN
await platform.invokeMethod('startVpn', {
  'dnsServer': '',  // empty = system DNS
  'domain': 'tunnel.example.com.',
});

// Stop VPN
await platform.invokeMethod('stopVpn');

// Get status
String status = await platform.invokeMethod('getStatus');
```

**Kotlin → Go:**
```kotlin
// Start tunnel
val error = Internal.startTunnel(dnsServer, domain)

// Stop tunnel
Internal.stopTunnel()

// Check status
val running = Internal.isRunning()
```

## License

Same as parent Aura project.

## See Also

- [FLUTTER-BUILD.md](../FLUTTER-BUILD.md) - Complete build guide
- [COMPLETE-ARCHITECTURE.md](../COMPLETE-ARCHITECTURE.md) - System architecture
- [ANDROID-BUILD.md](../ANDROID-BUILD.md) - Android-specific details
