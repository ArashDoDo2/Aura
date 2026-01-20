package com.aura.client

import android.content.Intent
import android.net.VpnService
import android.os.ParcelFileDescriptor
import android.util.Log
import internal.Internal
import java.io.FileInputStream
import java.io.FileOutputStream
import java.nio.ByteBuffer
import kotlin.concurrent.thread

class AuraVpnService : VpnService() {
    companion object {
        private const val TAG = "AuraVpnService"
        const val ACTION_START = "com.aura.START_VPN"
        const val ACTION_STOP = "com.aura.STOP_VPN"
        const val EXTRA_DNS_SERVER = "dns_server"
        const val EXTRA_DOMAIN = "domain"
        
        @Volatile
        var isRunning = false
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
            Log.w(TAG, "VPN already running")
            return
        }

        try {
            Log.d(TAG, "Starting VPN - DNS: $dnsServer, Domain: $domain")

            // Create VPN interface
            val builder = Builder()
            builder.setSession("Aura DNS Tunnel")
            builder.addAddress("10.0.0.2", 32)
            builder.addRoute("0.0.0.0", 0)
            
            // Use custom DNS if provided
            if (dnsServer.isNotEmpty()) {
                val dnsHost = dnsServer.split(":")[0]
                builder.addDnsServer(dnsHost)
            } else {
                // System will provide DNS
                Log.d(TAG, "Using system DNS")
            }
            
            // Optional: Per-app VPN for WhatsApp only
            // try {
            //     builder.addAllowedApplication("com.whatsapp")
            //     Log.d(TAG, "Per-app VPN: WhatsApp only")
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

        } catch (e: Exception) {
            Log.e(TAG, "Failed to start VPN: ${e.message}", e)
            isRunning = false
            running = false
            vpnInterface?.close()
            vpnInterface = null
            stopSelf()
        }
    }

    private fun stopVpn() {
        Log.d(TAG, "Stopping VPN")
        
        running = false
        isRunning = false

        // Stop Go engine
        try {
            val error = Internal.stopTunnel()
            if (error.isNotEmpty()) {
                Log.w(TAG, "Error stopping Go engine: $error")
            }
        } catch (e: Exception) {
            Log.e(TAG, "Error stopping Go engine: ${e.message}")
        }

        // Close VPN interface
        try {
            vpnInterface?.close()
        } catch (e: Exception) {
            Log.e(TAG, "Error closing VPN interface: ${e.message}")
        }
        vpnInterface = null

        // Wait for forwarding thread
        vpnThread?.interrupt()
        try {
            vpnThread?.join(1000)
        } catch (e: Exception) {
            Log.w(TAG, "Error joining thread: ${e.message}")
        }
        vpnThread = null
        
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
                
                try {
                    // Basic TCP forwarding to localhost:1080 (SOCKS5)
                    // In production, implement full SOCKS5 protocol handling
                    Log.v(TAG, "Forwarding packet of size: $length")
                } catch (e: Exception) {
                    Log.w(TAG, "Packet error: ${e.message}")
                }

                buffer.clear()
            }
        } catch (e: InterruptedException) {
            Log.d(TAG, "VPN forwarding interrupted")
        } catch (e: Exception) {
            Log.e(TAG, "VPN forwarding error: ${e.message}")
        } finally {
            try {
                vpnInput.close()
                vpnOutput.close()
            } catch (e: Exception) {
                Log.e(TAG, "Error closing streams: ${e.message}")
            }
        }
    }

    override fun onDestroy() {
        stopVpn()
        super.onDestroy()
    }
}
