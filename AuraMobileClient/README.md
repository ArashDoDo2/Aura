# Aura Native Android Client

Native Android app for Aura DNS Tunnel - no Flutter dependencies, direct Kotlin implementation.

## Features

✅ **Material Design UI**
✅ **Native Android (Kotlin)**
✅ **VpnService Integration**
✅ **System DNS Auto-Detection**
✅ **Real-time Status Display**
✅ **Direct Go Engine Integration via gomobile**

## Quick Start

### Prerequisites

```bash
# Android SDK (API 21+)
# Android NDK
# Go 1.21+
# gomobile
```

### Build Steps

#### 1. Build Go AAR

```powershell
cd C:\dev\Aura\Aura
gomobile bind -target=android/arm64,android/amd64 -o aura.aar ./internal
```

#### 2. Copy AAR to Project

```powershell
Copy-Item aura.aar AuraMobileClient\app\libs\
```

#### 3. Build Android App

```powershell
cd AuraMobileClient
./gradlew build
./gradlew installDebug
```

## Usage

1. **Enter Domain** (required): `tunnel.example.com.` with trailing dot
2. **Enter DNS** (optional): Leave empty for system DNS
3. **Tap CONNECT**: Grant VPN permission when prompted
4. **Status Display**: Shows connection status and configuration
5. **Tap DISCONNECT**: Stop VPN

## Architecture

```
UI (Kotlin/Android)
    ↕️
VpnService (Kotlin)
    ↕️
Go Engine (Internal)
    ↕️
SOCKS5 Proxy (127.0.0.1:1080)
```

## Key Files

- **MainActivity.kt**: UI controls, VPN permission, status display
- **AuraVpnService.kt**: VPN service + Go engine bridge
- **activity_main.xml**: UI layout with Material Design
- **build.gradle**: Dependencies including aura.aar

## Configuration

### Per-App VPN (WhatsApp Only)

Edit `AuraVpnService.kt` and uncomment:

```kotlin
try {
    builder.addAllowedApplication("com.whatsapp")
} catch (e: Exception) {
    Log.w(TAG, "Could not set per-app VPN: ${e.message}")
}
```

## Troubleshooting

### AAR Missing
```powershell
gomobile bind -target=android -o aura.aar ./internal
Copy-Item aura.aar AuraMobileClient\app\libs\
```

### Build Issues
```powershell
./gradlew clean
./gradlew build
```

### VPN Won't Connect
- Check domain ends with dot (.)
- Grant VPN permission
- View logs: `adb logcat -s AuraVpnService`

## Development

### Add Features
1. Edit `res/layout/activity_main.xml` for UI
2. Edit `MainActivity.kt` for logic
3. Edit `AuraVpnService.kt` for VPN integration

### Debug
```powershell
adb logcat -s AuraVpnService
```

## Permissions

```xml
<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.BIND_VPN_SERVICE" />
<uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
```

## See Also

- [COMPLETE-ARCHITECTURE.md](../COMPLETE-ARCHITECTURE.md)
- [ANDROID-BUILD.md](../ANDROID-BUILD.md)
- [FLUTTER-BUILD.md](../FLUTTER-BUILD.md)
