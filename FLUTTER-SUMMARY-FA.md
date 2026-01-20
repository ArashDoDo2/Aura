# پل Flutter + Go برای Aura - خلاصه پیاده‌سازی

## ✅ چه چیزی ساخته شد؟

### 1. رابط کاربری Flutter (UI)
**محل**: `flutter_aura/lib/`

- **main.dart**: صفحه اصلی با کنترل‌های VPN
  - دکمه Connect/Disconnect
  - ورودی DNS Server (اختیاری - خالی = DNS سیستم)
  - ورودی Domain
  - نمایش وضعیت اتصال

- **vpn_manager.dart**: پل ارتباطی با Kotlin
  - `startAura(dns, domain)` - شروع تونل
  - `stopAura()` - توقف تونل
  - `getStatus()` - وضعیت فعلی

### 2. پل Native اندروید (Kotlin)
**محل**: `flutter_aura/android/app/src/main/kotlin/`

- **MainActivity.kt**: مدیریت MethodChannel
  - دریافت فراخوانی از Flutter
  - درخواست مجوز VPN
  - راه‌اندازی سرویس

- **AuraVpnService.kt**: سرویس VPN + موتور Go
  - ایجاد رابط VPN (tun0)
  - فراخوانی `Internal.startTunnel()` از AAR
  - مدیریت Packet Forwarding
  - گزینه VPN فقط برای واتس‌اپ

### 3. API موتور Go
**محل**: `internal/mobile.go`

توابع جدید سازگار با gomobile:
```go
StartTunnel(dnsServer, domain string) string  // خطا یا خالی
StopTunnel() string
IsRunning() bool
GetStatus() string
```

### 4. فایل‌های پیکربندی

- **AndroidManifest.xml**: مجوزهای INTERNET و BIND_VPN_SERVICE
- **build.gradle**: وابستگی به `aura.aar`
- **settings.gradle**: پیکربندی پلاگین‌ها

## 🏗️ معماری سیستم

```
Flutter (Dart)
    ↕️ MethodChannel: "com.aura.proxy/vpn"
Android Native (Kotlin)
    ↕️ JNI (gomobile)
موتور Go (Aura)
    ↕️ SOCKS5 (localhost:1080)
تونل DNS
```

## 🔨 نحوه ساخت و اجرا

### گام 1: ساخت AAR از Go

```powershell
cd C:\dev\Aura\Aura
gomobile bind -target=android -o aura.aar ./internal
```

خروجی: `aura.aar` و `aura-sources.jar`

### گام 2: کپی AAR به پروژه Flutter

```powershell
New-Item -ItemType Directory -Force flutter_aura\android\app\libs
Copy-Item aura.aar flutter_aura\android\app\libs\
```

### گام 3: ساخت برنامه Flutter

```powershell
cd flutter_aura
flutter pub get
flutter build apk --release
```

خروجی: `build/app/outputs/flutter-apk/app-release.apk`

### گام 4: نصب روی گوشی

```powershell
flutter run
# یا
adb install build/app/outputs/flutter-apk/app-release.apk
```

## 📱 نحوه استفاده

1. **باز کردن برنامه**: "Aura DNS Tunnel"
2. **تنظیمات**:
   - DNS Server را خالی بگذارید (توصیه می‌شود)
   - Domain سرور خود را وارد کنید با نقطه پایانی
3. **اتصال**: روی دکمه CONNECT کلیک
4. **مجوز VPN**: مجوز را بدهید
5. **واتس‌اپ**: از واتس‌اپ استفاده کنید - ترافیک از طریق Aura عبور می‌کند

## 🎯 ویژگی‌های اضافی

### VPN فقط برای واتس‌اپ

در `AuraVpnService.kt` خط زیر را فعال کنید:

```kotlin
try {
    builder.addAllowedApplication("com.whatsapp")
} catch (e: Exception) {
    Log.w(TAG, "Could not set per-app VPN: ${e.message}")
}
```

این کار باعث می‌شود فقط ترافیک واتس‌اپ از طریق VPN عبور کند.

## 📂 ساختار فایل‌ها

```
C:\dev\Aura\Aura\
├── internal/
│   └── mobile.go              # API های gomobile
├── flutter_aura/              # برنامه Flutter
│   ├── lib/
│   │   ├── main.dart          # رابط کاربری
│   │   └── vpn_manager.dart   # پل MethodChannel
│   ├── android/
│   │   └── app/
│   │       ├── libs/
│   │       │   └── aura.aar   # کتابخانه Go (اینجا کپی کنید)
│   │       └── src/main/kotlin/com/aura/flutter_aura/
│   │           ├── MainActivity.kt      # هندلر MethodChannel
│   │           └── AuraVpnService.kt    # سرویس VPN
│   └── README.md
├── FLUTTER-BUILD.md           # راهنمای کامل ساخت
└── aura.aar                   # کتابخانه ساخته شده
```

## 🐛 رفع مشکلات رایج

### gomobile کار نمی‌کند

```powershell
gomobile init
gomobile clean
gomobile bind -target=android -o aura.aar ./internal
```

### Flutter build خطا می‌دهد

```powershell
flutter clean
flutter pub get
flutter doctor -v
flutter build apk --debug
```

### VPN وصل نمی‌شود

1. مجوز VPN داده شده؟
2. فایل AAR در `libs/` هست؟
3. Domain با نقطه (.) تمام می‌شود؟
4. لاگ‌ها را چک کنید:
   ```powershell
   flutter logs
   adb logcat | Select-String "Aura"
   ```

## 📚 اسناد بیشتر

- [FLUTTER-BUILD.md](../FLUTTER-BUILD.md): راهنمای کامل انگلیسی
- [COMPLETE-ARCHITECTURE.md](../COMPLETE-ARCHITECTURE.md): معماری کامل سیستم
- [.github/copilot-instructions.md](../.github/copilot-instructions.md): راهنمای AI Agent

## ✨ نتیجه

تمام کدها commit و push شدند:
- Commit: `0ad37c9`
- شاخه: `main`
- 16 فایل جدید
- 1945+ خط کد

سیستم Flutter + Go به طور کامل پیاده‌سازی شد و آماده استفاده است! 🚀
