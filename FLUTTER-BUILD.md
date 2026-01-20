# Flutter + Go Bridge for Aura DNS Tunnel

This guide explains how to build and run the complete Flutter app with Go engine integration.

## 🏗️ Architecture

```
Flutter UI (Dart)
    ↕ MethodChannel
Android Native (Kotlin)
    ↕ JNI (gomobile)
Go Engine (Aura)
```

## 📋 Prerequisites

### Required Tools
- **Go**: 1.21+ ([download](https://go.dev/dl/))
- **Flutter**: 3.0+ ([install guide](https://flutter.dev/docs/get-started/install))
- **Android Studio** with:
  - Android SDK (API 21+)
  - Android NDK (r23+)
  - Kotlin plugin
- **gomobile**: `go install golang.org/x/mobile/cmd/gomobile@latest`

### Environment Setup
```bash
# Initialize gomobile
gomobile init

# Set Android paths (Windows)
$env:ANDROID_HOME = "C:\Users\YourName\AppData\Local\Android\Sdk"
$env:ANDROID_NDK_HOME = "$env:ANDROID_HOME\ndk\25.2.9519653"

# Verify installations
go version
flutter --version
gomobile version
```

## 🔨 Build Steps

### Step 1: Build Go Engine (.aar)

```powershell
# Navigate to project root
cd C:\dev\Aura\Aura

# Build Android library
gomobile bind -target=android/arm64,android/amd64 -o aura.aar ./internal

# Verify output
ls aura.aar, aura-sources.jar
```

**Output files:**
- `aura.aar` - Android library (contains Go code)
- `aura-sources.jar` - Source mappings for debugging

### Step 2: Copy AAR to Flutter Project

```powershell
# Create libs directory
New-Item -ItemType Directory -Force flutter_aura\android\app\libs

# Copy AAR
Copy-Item aura.aar flutter_aura\android\app\libs\

# Verify
ls flutter_aura\android\app\libs\aura.aar
```

### Step 3: Build Flutter App

```powershell
# Navigate to Flutter project
cd flutter_aura

# Get dependencies
flutter pub get

# Build APK
flutter build apk --release

# Or build for debugging
flutter build apk --debug

# Output: build/app/outputs/flutter-apk/app-release.apk
```

## 🚀 Running the App

### Option 1: Run on Connected Device

```powershell
# List devices
flutter devices

# Run on device
flutter run

# Or install APK directly
adb install build/app/outputs/flutter-apk/app-release.apk
```

### Option 2: Run in Debug Mode

```powershell
# Hot reload enabled
flutter run --debug

# With verbose logging
flutter run -v
```

## 🧪 Testing

### Test Go Engine Separately

```powershell
# Build and test client
go build -o aura-client ./cmd/client
./aura-client -dns 1.1.1.1:53 -domain tunnel.example.com.

# Verify SOCKS5 proxy
Test-NetConnection -ComputerName 127.0.0.1 -Port 1080
```

### Test Flutter UI

```powershell
cd flutter_aura

# Run tests
flutter test

# Run integration tests (if available)
flutter drive --target=test_driver/app.dart
```

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
