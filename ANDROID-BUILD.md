# Building Aura Client for Android

This document provides instructions for compiling the Aura client for Android.

## Prerequisites

1. Install Go (1.20+)
2. Install gomobile:
```bash
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
```

## Testing on Android via Termux

Before compiling to .aar, test the client directly on Android:

1. Install Termux from F-Droid
2. In Termux, install Go:
```bash
pkg install golang git
```

3. Clone and run:
```bash
git clone https://github.com/ArashDoDo2/Aura
cd Aura
go run cmd/aura-client.go --dns YOUR_SERVER_IP:53 --domain aura.net.
```

4. Configure WhatsApp to use proxy `127.0.0.1:1080`

## Compiling to Android Archive (.aar)

To use Aura in an Android app with a GUI:

### Build the .aar library:

```bash
cd /workspaces/Aura
gomobile bind -target=android -o aura.aar github.com/ArashDoDo2/Aura/internal
```

This creates `aura.aar` and `aura-sources.jar`.

### Use in Android Studio:

1. Copy `aura.aar` to `app/libs/` in your Android project
2. Add to `app/build.gradle`:
```gradle
dependencies {
    implementation files('libs/aura.aar')
}
```

3. In your Activity or VPN Service:
```kotlin
import internal.Internal

// Start client
try {
    Internal.startAuraClient("8.8.8.8:53", "aura.net.")
    Log.d("Aura", "Client started")
} catch (e: Exception) {
    Log.e("Aura", "Failed to start: ${e.message}")
}

// Stop client
Internal.stopAuraClient()
```

## Graceful Shutdown

The client automatically handles interrupts (Ctrl+C in Termux) and can be stopped programmatically via `StopAuraClient()` from Android.

## Logging for Android

All errors are returned to the caller. Use Android's Logcat to capture any issues:
```bash
adb logcat | grep Aura
```

## Notes

- The client listens on port 1080 (SOCKS5)
- Ensure your Android app has INTERNET permission in `AndroidManifest.xml`
- For VPN-based routing, implement VpnService in your Android app
- The `.aar` exports `StartAuraClient(dnsServer, domain)` and `StopAuraClient()`
