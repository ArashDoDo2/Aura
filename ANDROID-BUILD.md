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

// Option 1: Use system DNS (recommended for mobile)
// Let the device use its configured DNS resolver
try {
    Internal.startAuraClient(
        "",                    // Empty = system resolver
        "tunnel.example.com.", // Your domain
        1080                   // SOCKS5 port
    )
    Log.d("Aura", "Client started with system DNS")
} catch (e: Exception) {
    Log.e("Aura", "Failed to start: ${e.message}")
}

// Option 2: Use specific DNS server
try {
    Internal.startAuraClient(
        "1.1.1.1:53",         // Custom DNS
        "tunnel.example.com.", // Your domain
        1080                   // SOCKS5 port
    )
    Log.d("Aura", "Client started with custom DNS")
} catch (e: Exception) {
    Log.e("Aura", "Failed to start: ${e.message}")
}

// Stop client
Internal.stopAuraClient()
```

### Android UI Example

For a user-friendly GUI, make DNS server field optional:

```kotlin
// UI Code
val dnsInput = findViewById<EditText>(R.id.dnsServerInput)
val domainInput = findViewById<EditText>(R.id.domainInput)
val portInput = findViewById<EditText>(R.id.portInput)

findViewById<Button>(R.id.startButton).setOnClickListener {
    val dns = dnsInput.text.toString().trim()  // User can leave empty
    val domain = domainInput.text.toString()
    val port = portInput.text.toString().toIntOrNull() ?: 1080
    
    try {
        Internal.startAuraClient(dns, domain, port)
        Toast.makeText(this, 
            if (dns.isEmpty()) "Using system DNS" else "Using $dns",
            Toast.LENGTH_SHORT
        ).show()
    } catch (e: Exception) {
        Toast.makeText(this, "Error: ${e.message}", Toast.LENGTH_LONG).show()
    }
}

findViewById<Button>(R.id.stopButton).setOnClickListener {
    Internal.stopAuraClient()
    Toast.makeText(this, "Stopped", Toast.LENGTH_SHORT).show()
}
```

## Graceful Shutdown

The client automatically handles interrupts (Ctrl+C in Termux) and can be stopped programmatically via `StopAuraClient()` from Android.

## Logging for Android

All errors are returned to the caller. Use Android's Logcat to capture any issues:
```bash
adb logcat | grep Aura
```

## Notes

- The client listens on port 1080 (SOCKS5) by default
- DNS server is now **optional** - leave empty to use device's DNS
- Ensure your Android app has INTERNET permission in `AndroidManifest.xml`
- For VPN-based routing, implement VpnService in your Android app
- The `.aar` exports `StartAuraClient(dnsServer, domain, port)` and `StopAuraClient()`
- System DNS is perfect for mobile networks where DNS changes dynamically
