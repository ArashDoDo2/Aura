package com.aura.client

import android.content.Intent
import android.net.VpnService
import android.os.Bundle
import android.text.Editable
import android.text.TextWatcher
import android.widget.Button
import android.widget.EditText
import android.widget.ImageView
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import com.google.android.material.snackbar.Snackbar
import internal.Internal

class MainActivity : AppCompatActivity() {
    private val VPN_REQUEST_CODE = 100
    
    // UI Components
    private lateinit var dnsInput: EditText
    private lateinit var domainInput: EditText
    private lateinit var connectBtn: Button
    private lateinit var statusIcon: ImageView
    private lateinit var statusText: TextView
    private lateinit var statusDetails: TextView
    
    private var isConnected = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
        
        initializeViews()
        setupListeners()
        updateUI()
    }

    private fun initializeViews() {
        dnsInput = findViewById(R.id.dnsInput)
        domainInput = findViewById(R.id.domainInput)
        connectBtn = findViewById(R.id.connectBtn)
        statusIcon = findViewById(R.id.statusIcon)
        statusText = findViewById(R.id.statusText)
        statusDetails = findViewById(R.id.statusDetails)
        
        // Set default values
        dnsInput.hint = "Leave empty for system DNS"
        domainInput.hint = "e.g., tunnel.example.com."
    }

    private fun setupListeners() {
        connectBtn.setOnClickListener {
            if (isConnected) {
                disconnectVpn()
            } else {
                connectVpn()
            }
        }
        
        // Update UI when input changes
        dnsInput.addTextChangedListener(object : TextWatcher {
            override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) {}
            override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {}
            override fun afterTextChanged(s: Editable?) { updateUI() }
        })
        
        domainInput.addTextChangedListener(object : TextWatcher {
            override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) {}
            override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {}
            override fun afterTextChanged(s: Editable?) { updateUI() }
        })
    }

    private fun connectVpn() {
        val dns = dnsInput.text.toString().trim()
        val domain = domainInput.text.toString().trim()
        
        if (domain.isEmpty()) {
            showSnackbar("Domain cannot be empty", true)
            return
        }
        
        if (!domain.endsWith(".")) {
            showSnackbar("Domain must end with dot (.)", true)
            return
        }
        
        // Request VPN permission
        val intent = VpnService.prepare(this)
        if (intent != null) {
            startActivityForResult(intent, VPN_REQUEST_CODE)
        } else {
            startVpnService(dns, domain)
        }
    }

    private fun disconnectVpn() {
        val intent = Intent(this, AuraVpnService::class.java)
        intent.action = AuraVpnService.ACTION_STOP
        startService(intent)
        
        isConnected = false
        updateUI()
        showSnackbar("VPN Stopped", false)
    }

    private fun startVpnService(dns: String, domain: String) {
        val intent = Intent(this, AuraVpnService::class.java)
        intent.action = AuraVpnService.ACTION_START
        intent.putExtra(AuraVpnService.EXTRA_DNS_SERVER, dns)
        intent.putExtra(AuraVpnService.EXTRA_DOMAIN, domain)
        startService(intent)
        
        isConnected = true
        updateUI()
        showSnackbar("VPN Starting...", false)
    }

    private fun updateUI() {
        val isValid = domainInput.text.toString().trim().isNotEmpty()
        connectBtn.isEnabled = isValid
        connectBtn.text = if (isConnected) "DISCONNECT" else "CONNECT"
        connectBtn.setBackgroundColor(
            if (isConnected) 
                resources.getColor(R.color.disconnect_color, theme)
            else 
                resources.getColor(R.color.connect_color, theme)
        )
        
        // Update status display
        if (isConnected) {
            statusIcon.setImageResource(android.R.drawable.ic_dialog_check)
            statusText.text = "Connected"
            statusText.setTextColor(resources.getColor(R.color.status_connected, theme))
            val dns = if (dnsInput.text.isEmpty()) "System DNS" else dnsInput.text.toString()
            statusDetails.text = "Domain: ${domainInput.text}\nDNS: $dns"
        } else {
            statusIcon.setImageResource(android.R.drawable.ic_dialog_info)
            statusText.text = "Disconnected"
            statusText.setTextColor(resources.getColor(R.color.status_disconnected, theme))
            statusDetails.text = "Configure and connect to start tunneling"
        }
    }

    private fun showSnackbar(message: String, isError: Boolean) {
        Snackbar.make(
            connectBtn,
            message,
            Snackbar.LENGTH_SHORT
        ).setBackgroundTint(
            if (isError) resources.getColor(R.color.error_color, theme)
            else resources.getColor(R.color.success_color, theme)
        ).show()
    }

    @Deprecated("Deprecated in Java")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        
        if (requestCode == VPN_REQUEST_CODE) {
            if (resultCode == RESULT_OK) {
                val dns = dnsInput.text.toString().trim()
                val domain = domainInput.text.toString().trim()
                startVpnService(dns, domain)
            } else {
                showSnackbar("VPN permission denied", true)
            }
        }
    }

    override fun onResume() {
        super.onResume()
        isConnected = Internal.isRunning()
        updateUI()
    }
}
