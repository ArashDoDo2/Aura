import 'dart:ffi' as ffi;
import 'dart:io' show Platform;

import 'package:ffi/ffi.dart' as pkg_ffi;

class AuraBridge {
  AuraBridge._();

  static final ffi.DynamicLibrary _lib = _openLib();

  static ffi.DynamicLibrary _openLib() {
    if (Platform.isAndroid) {
      return ffi.DynamicLibrary.open('libaura.so');
    } else if (Platform.isWindows) {
      return ffi.DynamicLibrary.open('aura.dll');
    } else if (Platform.isLinux) {
      return ffi.DynamicLibrary.open('libaura.so');
    } else if (Platform.isMacOS) {
      return ffi.DynamicLibrary.open('libaura.dylib');
    }
    throw UnsupportedError('Unsupported platform: ${Platform.operatingSystem}');
  }

  // C signatures
  static final _startAura = _lib.lookupFunction<
      ffi.Void Function(ffi.Pointer<ffi.Char>, ffi.Pointer<ffi.Char>),
      void Function(ffi.Pointer<ffi.Char>, ffi.Pointer<ffi.Char>)>('StartAura');

  static final _stopAura =
      _lib.lookupFunction<ffi.Void Function(), void Function()>('StopAura');

  /// Start Aura tunnel. `dns` can be empty for system DNS.
  static void startAura(String dns, String domain) {
    final dnsPtr = dns.toNativeUtf8().cast<ffi.Char>();
    final domainPtr = domain.toNativeUtf8().cast<ffi.Char>();
    try {
      _startAura(dnsPtr, domainPtr);
    } finally {
      pkg_ffi.malloc.free(dnsPtr);
      pkg_ffi.malloc.free(domainPtr);
    }
  }

  /// Stop Aura tunnel.
  static void stopAura() {
    _stopAura();
  }
}
