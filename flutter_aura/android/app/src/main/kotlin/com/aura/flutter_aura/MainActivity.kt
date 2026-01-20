package com.aura.flutter_aura

import android.app.Activity
import android.content.Intent
import android.net.VpnService
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private val CHANNEL = "com.aura.proxy/vpn"
    private val VPN_REQUEST_CODE = 1

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL).setMethodCallHandler { call, result ->
            when (call.method) {
                "startVpn" -> {
                    val dnsServer = call.argument<String>("dnsServer") ?: ""
                    val domain = call.argument<String>("domain") ?: ""
                    startVpn(dnsServer, domain, result)
                }
                "stopVpn" -> {
                    stopVpn(result)
                }
                "getStatus" -> {
                    val status = if (AuraVpnService.isRunning) "running" else "stopped"
                    result.success(status)
                }
                else -> {
                    result.notImplemented()
                }
            }
        }
    }

    private fun startVpn(dnsServer: String, domain: String, result: MethodChannel.Result) {
        if (domain.isEmpty()) {
            result.error("INVALID_ARGS", "Domain cannot be empty", null)
            return
        }

        // Request VPN permission
        val intent = VpnService.prepare(applicationContext)
        if (intent != null) {
            // Store result for later use
            AuraVpnService.result = result
            startActivityForResult(intent, VPN_REQUEST_CODE)
        } else {
            // Permission already granted
            startVpnService(dnsServer, domain, result)
        }
    }

    private fun stopVpn(result: MethodChannel.Result) {
        AuraVpnService.result = result
        val intent = Intent(this, AuraVpnService::class.java)
        intent.action = AuraVpnService.ACTION_STOP
        startService(intent)
    }

    private fun startVpnService(dnsServer: String, domain: String, result: MethodChannel.Result) {
        AuraVpnService.result = result
        val intent = Intent(this, AuraVpnService::class.java)
        intent.action = AuraVpnService.ACTION_START
        intent.putExtra(AuraVpnService.EXTRA_DNS_SERVER, dnsServer)
        intent.putExtra(AuraVpnService.EXTRA_DOMAIN, domain)
        startService(intent)
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        
        if (requestCode == VPN_REQUEST_CODE) {
            if (resultCode == Activity.RESULT_OK) {
                // VPN permission granted, but we need the arguments
                // This is handled by re-calling startVpn which will now succeed
                AuraVpnService.result?.success("")
            } else {
                // VPN permission denied
                AuraVpnService.result?.error("VPN_PERMISSION_DENIED", "User denied VPN permission", null)
            }
            AuraVpnService.result = null
        }
    }
}
