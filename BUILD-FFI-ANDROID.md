# Build libaura.so for Android (ARM64) via Go FFI

## Architecture

```
Flutter (Dart)
    ↕ Dart FFI
 libaura.so (Go C-shared library)
    ↕
 Go SOCKS5 proxy → WhatsApp:5222
```

## Prerequisites

- **Go**: 1.21+
- **Android NDK**: 25.2.9519653 or 29.0.14206865
- **Android SDK**: API 21+
- **Windows**: PowerShell 5.1+

## Step 1: Verify NDK Installation

```powershell
$env:ANDROID_HOME = "$env:LOCALAPPDATA\Android\Sdk"
Get-ChildItem "$env:ANDROID_HOME\ndk"
```

Note the NDK version (e.g., `29.0.14206865`).

## Step 2: Set Environment Variables

Replace with your NDK version:

```powershell
$env:ANDROID_HOME = "$env:LOCALAPPDATA\Android\Sdk"
$env:ANDROID_NDK_HOME = "$env:ANDROID_HOME\ndk\29.0.14206865"
$env:CC = "$env:ANDROID_NDK_HOME\toolchains\llvm\prebuilt\windows-x86_64\bin\aarch64-linux-android21-clang.cmd"
$env:CGO_ENABLED = "1"
$env:GOOS = "android"
$env:GOARCH = "arm64"
```

**Notes:**
- Use `.cmd` wrapper (not `.exe`) for NDK 25.2+
- API level can be 21, 24, or 26 (adjust filename accordingly)

## Step 3: Build libaura.so

```powershell
cd C:\dev\Aura\Aura
go build -buildmode=c-shared -o libaura.so ./bridge.go
```

**Output:** `libaura.so` (~7 MB) and `libaura.h`

## Step 4: Deploy to Flutter

```powershell
# Create jniLibs directory
New-Item -ItemType Directory -Force flutter_aura\android\app\src\main\jniLibs\arm64-v8a

# Copy .so file
Copy-Item libaura.so flutter_aura\android\app\src\main\jniLibs\arm64-v8a\
```

## Step 5: Build and Install APK

```powershell
cd flutter_aura

# Get dependencies (includes package:ffi)
flutter pub get

# Build debug APK
flutter build apk --debug

# Install on device
adb install -r build/app/outputs/flutter-apk/app-debug.apk
```

## Architecture Details

**bridge.go** exports two C functions:
- `StartAura(dnsServer *C.char, domain *C.char)` - Starts SOCKS5 proxy
- `StopAura()` - Stops proxy

**aura_bridge.dart** wraps them with:
- Platform-specific library loading (`.so` on Android, `.dll` on Windows)
- Safe C string allocation/deallocation
- Synchronous invocation (non-blocking in UI)

**main.dart** UI:
- DNS input (optional, empty = system DNS)
- Domain input (required)
- Connect/Disconnect button
- Status display

## Troubleshooting

### Compiler Not Found
```
error: C compiler "...clang.exe" not found
```
→ Use `.cmd` wrapper instead: `aarch64-linux-android21-clang.cmd`

### NDK Path Missing
```
error: no usable NDK
```
→ Verify NDK installed: `Get-ChildItem $env:ANDROID_HOME\ndk`

### APK Build Fails
```
Build failed due to use of deleted Android v1 embedding
```
→ Regenerate Android project:
```powershell
flutter clean
rm -r android
flutter create --platforms=android .
flutter pub get
flutter build apk --debug
```

## Testing on Android Device

1. **Connect device via USB** and enable USB debugging
2. **Install APK**:
   ```powershell
   adb install -r flutter_aura/build/app/outputs/flutter-apk/app-debug.apk
   ```
3. **Grant VPN permission** when prompted
4. **Enter configuration**:
   - DNS: Leave empty (system) or enter custom (e.g., `1.1.1.1:53`)
   - Domain: Your server domain (e.g., `tunnel.example.com.` with trailing dot)
5. **Tap Connect** and verify WhatsApp traffic flows through tunnel

## For Windows Desktop Testing

Build Windows version:

```powershell
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CC = "gcc"  # Or clang from NDK
go build -buildmode=c-shared -o aura.dll ./bridge.go
```

Dart FFI automatically loads `aura.dll` on Windows platforms.
