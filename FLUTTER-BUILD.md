# Flutter + Go Bridge for Aura DNS Tunnel (Dart FFI)

This guide explains how to build and run the complete Flutter app with Go engine integration via Dart FFI (avoiding NDK/gomobile issues).

## 🏗️ Architecture

```
Flutter UI (Dart)
    ↕ Dart FFI (dart:ffi)
  libaura.so (Go C-shared library)
    ↕
  Go SOCKS5 Proxy
    ↕
WhatsApp Server (port 5222)
```

**Why FFI instead of gomobile?**
- ✅ No NDK platform/API mismatch errors
- ✅ Direct C interop—faster, simpler
- ✅ No need for MethodChannel/VpnService
- ✅ Works on Windows, Linux, macOS for testing

## 📋 Prerequisites

### Required Tools
- **Go**: 1.21+ ([download](https://go.dev/dl/))
- **Flutter**: 3.0+ ([install guide](https://flutter.dev/docs/get-started/install))
- **Android NDK**: 25.2.9519653+ ([prebuilt recommended](https://developer.android.com/ndk/downloads))
- **Android SDK**: API 21+
- **Android Studio**: Recommended for device/emulator management

### Verify Installations
```powershell
go version
flutter --version
adb --version

# Check NDK
Get-ChildItem "$env:LOCALAPPDATA\Android\Sdk\ndk"
```

## 🔨 Build Steps

### Step 1: Build Go Shared Library

```powershell
# Set NDK env vars (adjust version as needed)
$env:ANDROID_HOME = "$env:LOCALAPPDATA\Android\Sdk"
$env:ANDROID_NDK_HOME = "$env:ANDROID_HOME\ndk\29.0.14206865"
$env:CC = "$env:ANDROID_NDK_HOME\toolchains\llvm\prebuilt\windows-x86_64\bin\aarch64-linux-android21-clang.cmd"
$env:CGO_ENABLED = "1"
$env:GOOS = "android"
$env:GOARCH = "arm64"

# Navigate to project root
cd C:\dev\Aura\Aura

# Build shared library
go build -buildmode=c-shared -o libaura.so ./bridge.go

# Verify output
Get-Item libaura.so | Select-Object Name, Length
```

**Output:** `libaura.so` (~7 MB) + `libaura.h` (C header)

### Step 2: Deploy .so to Flutter

```powershell
# Create jniLibs directory tree
New-Item -ItemType Directory -Force flutter_aura\android\app\src\main\jniLibs\arm64-v8a

# Copy .so file
Copy-Item libaura.so flutter_aura\android\app\src\main\jniLibs\arm64-v8a\
```

### Step 3: Build Flutter App

```powershell
# Navigate to Flutter project
cd flutter_aura

# Get dependencies (includes package:ffi)
flutter pub get

# Build APK
flutter build apk --release

# Or build for debugging
flutter build apk --debug

# Output: build/app/outputs/flutter-apk/app-debug.apk (143 MB)
```

## 🚀 Running the App

### Option 1: Install and Run

```powershell
# Install built APK
adb install -r build/app/outputs/flutter-apk/app-debug.apk

# Check logs
adb logcat -s Flutter
```

## 🧪 Testing

### Test Go Engine Separately

```powershell
# Build and test client
cd C:\dev\Aura\Aura
go build -o aura-client ./cmd/client

# Run with system DNS
.\aura-client -domain tunnel.example.com.

# Or with custom DNS
.\aura-client -dns 1.1.1.1:53 -domain tunnel.example.com.

# Verify SOCKS5 proxy on 127.0.0.1:1080
Test-NetConnection -ComputerName 127.0.0.1 -Port 1080
```

### Test Flutter App on Android Device

1. Start app and grant VPN permission
2. Enter domain (e.g., `tunnel.example.com.`)
3. Leave DNS empty (system) or custom (e.g., `1.1.1.1:53`)
4. Tap **Connect**
5. Open WhatsApp and send message
6. Check device logs: `adb logcat | Select-String 'Aura'`
7. Tap **Disconnect** to stop

## 🔧 Configuration

### Server Setup (Required)

Before using the app, you need an authoritative DNS server:

```powershell
# On server (Linux)
cd C:\dev\Aura\Aura
go build -o aura-server ./cmd/server
sudo ./aura-server -domain your-domain.com. -addr :53
```

### App Configuration

Edit values in Flutter app:
- **DNS Server**: Leave empty for system DNS (recommended)
- **Domain**: Your authoritative domain (e.g., `tunnel.example.com.`)

## 📱 Using the App

1. **Launch App**: Open "Aura DNS Tunnel"
2. **Configure**:
   - DNS Server: Leave empty or enter custom (e.g., `1.1.1.1:53`)
   - Domain: Enter your server domain with trailing dot
3. **Connect**: Tap "CONNECT" button
4. **VPN Permission**: Grant VPN permission when prompted
5. **Use WhatsApp**: Open WhatsApp - traffic routes through Aura

## 🐛 Troubleshooting

### gomobile build fails

```powershell
# Ensure gomobile is initialized
gomobile init

# Clean and rebuild
gomobile clean
gomobile bind -target=android -o aura.aar ./internal
```

### Flutter build fails

```powershell
# Clean build cache
flutter clean
flutter pub get

# Check Android SDK
flutter doctor -v

# Rebuild
flutter build apk --debug
```

### VPN won't connect

1. Check VPN permission granted
2. Verify AAR is in `libs/` folder
3. Check logs:
   ```powershell
   flutter logs
   # Or
   adb logcat | Select-String "Aura"
   ```
4. Verify domain format (must end with `.`)

### App crashes on start

```powershell
# Check native logs
adb logcat -s "AuraVpnService"

# Common issues:
# - Missing AAR file
# - Incorrect package name
# - Missing permissions in AndroidManifest.xml
```

## 📂 Project Structure

```
C:\dev\Aura\Aura\
├── internal/               # Go engine
│   ├── client.go          # SOCKS5 + DNS client
│   ├── server.go          # DNS server
│   └── mobile.go          # gomobile exports
├── flutter_aura/          # Flutter app
│   ├── lib/
│   │   ├── main.dart      # UI
│   │   └── vpn_manager.dart  # MethodChannel bridge
│   └── android/
│       └── app/
│           ├── libs/
│           │   └── aura.aar  # Go engine (copy here)
│           └── src/main/kotlin/
│               └── com/aura/flutter_aura/
│                   ├── MainActivity.kt      # MethodChannel handler
│                   └── AuraVpnService.kt    # VPN service
└── aura.aar               # Built Go library
```

## 🔒 Per-App VPN (WhatsApp Only)

To intercept only WhatsApp traffic, uncomment in `AuraVpnService.kt`:

```kotlin
try {
    builder.addAllowedApplication("com.whatsapp")
} catch (e: Exception) {
    Log.w(TAG, "Could not set per-app VPN: ${e.message}")
}
```

**Note**: Per-app VPN requires Android 5.0+ (API 21+)

## 📚 Additional Resources

- [gomobile documentation](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile)
- [Flutter platform channels](https://docs.flutter.dev/platform-integration/platform-channels)
- [Android VpnService](https://developer.android.com/reference/android/net/VpnService)
- [Aura Architecture](../COMPLETE-ARCHITECTURE.md)

## 🆘 Support

For issues:
1. Check [GitHub Issues](https://github.com/ArashDoDo2/Aura/issues)
2. Review [COMPLETE-ARCHITECTURE.md](../COMPLETE-ARCHITECTURE.md)
3. Check `flutter doctor` output
4. Verify Go version: `go version`

## 🎯 Next Steps

After successful build:
1. Deploy your DNS server
2. Configure domain in app
3. Test with WhatsApp
4. Monitor logs for debugging
5. Consider adding features:
   - Connection status notifications
   - Traffic statistics
   - Auto-reconnect
   - Multiple domain profiles
