import 'package:flutter/services.dart';

class VpnManager {
  static const platform = MethodChannel('com.aura.proxy/vpn');

  /// Start Aura DNS Tunnel
  /// [dnsServer] - DNS server address (empty = system resolver)
  /// [domain] - Target domain (e.g., "tunnel.example.com.")
  Future<void> startAura(String dnsServer, String domain) async {
    try {
      final result = await platform.invokeMethod('startVpn', {
        'dnsServer': dnsServer,
        'domain': domain,
      });
      
      if (result != null && result.toString().isNotEmpty) {
        throw Exception(result);
      }
    } on PlatformException catch (e) {
      throw Exception('Failed to start VPN: ${e.message}');
    }
  }

  /// Stop Aura DNS Tunnel
  Future<void> stopAura() async {
    try {
      final result = await platform.invokeMethod('stopVpn');
      
      if (result != null && result.toString().isNotEmpty) {
        throw Exception(result);
      }
    } on PlatformException catch (e) {
      throw Exception('Failed to stop VPN: ${e.message}');
    }
  }

  /// Get tunnel status
  Future<String> getStatus() async {
    try {
      final result = await platform.invokeMethod('getStatus');
      return result?.toString() ?? 'unknown';
    } on PlatformException catch (e) {
      return 'error: ${e.message}';
    }
  }
}
