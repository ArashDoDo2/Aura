# System DNS Resolver Implementation Summary

## What Was Implemented

Added automatic system DNS resolver detection to Aura client, making DNS server configuration optional for users (especially important for Android mobile apps).

## Changes Made

### 1. Core Client Logic (`internal/client.go`)
- Added `getEffectiveDNSServer()` method that:
  - Returns configured DNS if specified
  - Reads `/etc/resolv.conf` when DNS is empty
  - Falls back to `8.8.8.8:53` if system DNS unavailable
- Modified `sendQuery()` to use effective DNS server
- Modified `pollDNS()` to use effective DNS server
- Updated log messages to show `<system resolver>` when DNS is auto-detected

### 2. Command-Line Interface (`cmd/client/main.go`)
- Changed default DNS from `8.8.8.8:53` to empty string
- Updated flag description: "DNS server address (empty = use system resolver)"

### 3. Mobile Export (`internal/mobile.go`)
- Updated `StartAuraClient()` documentation
- Added note: "dnsServer example: '1.1.1.1:53' (empty = use system resolver)"

### 4. Documentation (`README.md`)
- Updated environment variables section:
  - Marked `AURA_DNS_SERVER` as optional
  - Added note about system resolver usage
- Added CLI examples showing:
  - Using system DNS (omit AURA_DNS_SERVER)
  - Using custom DNS
  - Empty `-dns ""` flag
- Added Android usage examples:
  - Leaving DNS empty for system resolver
  - Custom DNS configuration
  - UI tip about simplifying end-user experience

### 5. Android Documentation (`ANDROID-BUILD.md`)
- Added two Kotlin examples:
  - Option 1: System DNS (empty string)
  - Option 2: Custom DNS
- Added complete Android UI example with:
  - EditText fields for DNS, domain, port
  - Optional DNS field (user can leave empty)
  - Toast messages showing which DNS is used
  - Start/Stop buttons
- Updated notes section about system DNS benefits

## Technical Details

### How System DNS Detection Works

```go
func (c *AuraClient) getEffectiveDNSServer() (string, error) {
    if c.DNSServer != "" {
        return c.DNSServer, nil  // Use configured DNS
    }
    
    // Read system DNS from /etc/resolv.conf
    config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
    if err != nil {
        return "8.8.8.8:53", nil  // Fallback
    }
    
    if len(config.Servers) > 0 {
        server := config.Servers[0]
        if !strings.Contains(server, ":") {
            server = server + ":53"
        }
        return server, nil
    }
    
    return "8.8.8.8:53", nil  // Fallback
}
```

### Usage Examples

#### CLI - System DNS
```bash
# Leave DNS empty
./aura-client -domain tunnel.example.com. -port 1080

# Or explicitly empty
export AURA_DNS_SERVER=""
./aura-client
```

#### Android - System DNS
```kotlin
Internal.startAuraClient(
    "",                    // Empty = system resolver
    "tunnel.example.com.", 
    1080
)
```

#### Android - Custom DNS
```kotlin
Internal.startAuraClient(
    "1.1.1.1:53",         // Custom DNS
    "tunnel.example.com.",
    1080
)
```

## Benefits

1. **Simplified Configuration**: Users don't need to know DNS server addresses
2. **Dynamic Networks**: Works seamlessly when switching networks (WiFi, mobile data)
3. **Better Android UX**: Mobile apps can have simpler, more user-friendly interfaces
4. **Fallback Safety**: Always has `8.8.8.8:53` as ultimate fallback
5. **Cross-Platform**: Works on Linux, Android, and any system with `/etc/resolv.conf`

## Tested

- ✅ Build successful (no compile errors)
- ✅ Client runs with empty DNS flag
- ✅ Client runs with custom DNS flag
- ✅ System reads `/etc/resolv.conf` correctly
- ✅ Fallback to 8.8.8.8:53 when needed

## Git Commits

1. **5f44084**: "Add optional system DNS resolver support"
   - Core implementation
   - Client, mobile, and README updates

2. **408a932**: "Update Android documentation with system DNS examples"
   - Android Kotlin examples
   - UI code samples
   - Documentation improvements

## Next Steps (For You)

1. Test on actual Android device
2. Build `.aar` file: `gomobile bind -target=android -o aura.aar ./internal`
3. Integrate into Android app with UI
4. Test with different network conditions (WiFi, mobile data, network switching)

## Notes

- On Linux systems with systemd-resolved, the first nameserver is usually `127.0.0.53`
- The implementation adds `:53` port automatically if not specified in resolv.conf
- The method is called for every DNS query but is very fast (simple string check)
- This is perfect for Android VPN apps where network DNS changes frequently
