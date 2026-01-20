package com.aura.flutter_aura

import android.content.Intent
import android.net.VpnService
import android.os.ParcelFileDescriptor
import android.util.Log
import io.flutter.plugin.common.MethodChannel
import internal.Internal
import java.io.FileInputStream
import java.io.FileOutputStream
import java.net.InetSocketAddress
import java.nio.ByteBuffer
import java.nio.channels.DatagramChannel
import java.nio.channels.SocketChannel
import kotlin.concurrent.thread

class AuraVpnService : VpnService() {
    companion object {
        private const val TAG = "AuraVpnService"
        private const val VPN_ADDRESS = "10.0.0.2"
        private const val VPN_ROUTE = "0.0.0.0"
        const val ACTION_START = "com.aura.START_VPN"
        const val ACTION_STOP = "com.aura.STOP_VPN"
        const val EXTRA_DNS_SERVER = "dns_server"
        const val EXTRA_DOMAIN = "domain"
        
        @Volatile
        var isRunning = false
        
        @Volatile
        var result: MethodChannel.Result? = null
    }

    private var vpnInterface: ParcelFileDescriptor? = null
    private var vpnThread: Thread? = null
    @Volatile
    private var running = false

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START -> {
                val dnsServer = intent.getStringExtra(EXTRA_DNS_SERVER) ?: ""
                val domain = intent.getStringExtra(EXTRA_DOMAIN) ?: ""
                startVpn(dnsServer, domain)
            }
            ACTION_STOP -> {
                stopVpn()
            }
        }
        return START_STICKY
    }

    private fun startVpn(dnsServer: String, domain: String) {
        if (isRunning) {
            result?.success("VPN already running")
            result = null
            return
        }

        try {
            Log.d(TAG, "Starting VPN - DNS: $dnsServer, Domain: $domain")

            // Create VPN interface
            val builder = Builder()
            builder.setSession("Aura DNS Tunnel")
            builder.addAddress(VPN_ADDRESS, 32)
            builder.addRoute(VPN_ROUTE, 0)
            
            // Use custom DNS if provided, otherwise system default
            if (dnsServer.isNotEmpty()) {
                val dnsHost = dnsServer.split(":")[0]
                builder.addDnsServer(dnsHost)
            } else {
                builder.addDnsServer("8.8.8.8")
            }
            
            // Optional: Intercept only WhatsApp traffic
            // Uncomment to enable per-app VPN
            // try {
            //     builder.addAllowedApplication("com.whatsapp")
            // } catch (e: Exception) {
            //     Log.w(TAG, "Could not set per-app VPN: ${e.message}")
            // }

            vpnInterface = builder.establish()
            if (vpnInterface == null) {
                throw Exception("Failed to establish VPN interface")
            }

            // Start Aura Go engine
            val error = Internal.startTunnel(dnsServer, domain)
            if (error.isNotEmpty()) {
                vpnInterface?.close()
                vpnInterface = null
                throw Exception(error)
            }

            isRunning = true
            running = true

            // Start packet forwarding thread
            vpnThread = thread(start = true, name = "VpnForwarder") {
                forwardPackets()
            }

            Log.i(TAG, "VPN started successfully")
            result?.success("")
            result = null

        } catch (e: Exception) {
            Log.e(TAG, "Failed to start VPN", e)
            isRunning = false
            running = false
            vpnInterface?.close()
            vpnInterface = null
            result?.error("VPN_ERROR", e.message, null)
            result = null
            stopSelf()
        }
    }

    private fun stopVpn() {
        Log.d(TAG, "Stopping VPN")
        
        running = false
        isRunning = false

        // Stop Go engine
        try {
            Internal.stopTunnel()
        } catch (e: Exception) {
            Log.e(TAG, "Error stopping Go engine", e)
        }

        // Close VPN interface
        try {
            vpnInterface?.close()
        } catch (e: Exception) {
            Log.e(TAG, "Error closing VPN interface", e)
        }
        vpnInterface = null

        // Wait for forwarding thread
        vpnThread?.interrupt()
        vpnThread?.join(1000)
        vpnThread = null

        result?.success("")
        result = null
        
        Log.i(TAG, "VPN stopped")
        stopSelf()
    }

    private fun forwardPackets() {
        val vpnInput = FileInputStream(vpnInterface!!.fileDescriptor)
        val vpnOutput = FileOutputStream(vpnInterface!!.fileDescriptor)
        val buffer = ByteBuffer.allocate(32767)

        try {
            while (running && !Thread.currentThread().isInterrupted) {
                // Read packet from VPN interface
                val length = vpnInput.read(buffer.array())
                if (length <= 0) {
                    Thread.sleep(10)
                    continue
                }

                buffer.limit(length)
                
                // Simple packet forwarding to SOCKS5 proxy
                // In a production app, you'd implement proper SOCKS5 protocol handling
                // or use a library like Shadowsocks/V2Ray core for packet forwarding
                
                try {
                    // Basic TCP forwarding to localhost:1080 (SOCKS5)
                    // This is a simplified implementation
                    // For production, use a proper VPN packet handler
                    forwardToSocks5(buffer)
                } catch (e: Exception) {
                    Log.w(TAG, "Packet forwarding error: ${e.message}")
                }

                buffer.clear()
            }
        } catch (e: InterruptedException) {
            Log.d(TAG, "VPN forwarding interrupted")
        } catch (e: Exception) {
            Log.e(TAG, "VPN forwarding error", e)
        } finally {
            try {
                vpnInput.close()
                vpnOutput.close()
            } catch (e: Exception) {
                Log.e(TAG, "Error closing streams", e)
            }
        }
    }

    private fun forwardToSocks5(packet: ByteBuffer) {
        // TODO: Implement proper SOCKS5 packet forwarding
        // This is a placeholder for the actual implementation
        // You would need to:
        // 1. Parse IP packet
        // 2. Extract destination
        // 3. Create SOCKS5 connection
        // 4. Forward data bidirectionally
        
        // For now, this just logs the packet size
        Log.v(TAG, "Forwarding packet of size: ${packet.remaining()}")
    }

    override fun onDestroy() {
        stopVpn()
        super.onDestroy()
    }
}
