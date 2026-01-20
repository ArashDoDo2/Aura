package com.example.auramobileclient

import android.os.Bundle
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity

class MainActivity : AppCompatActivity() {
    private var isConnected = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        val dnsEdit = findViewById<EditText>(R.id.dnsEdit)
        val sessionEdit = findViewById<EditText>(R.id.sessionEdit)
        val statusText = findViewById<TextView>(R.id.statusText)
        val connectBtn = findViewById<Button>(R.id.connectBtn)

        connectBtn.setOnClickListener {
            isConnected = !isConnected
            if (isConnected) {
                statusText.text = getString(R.string.connected)
                connectBtn.text = getString(R.string.disconnect)
            } else {
                statusText.text = getString(R.string.disconnected)
                connectBtn.text = getString(R.string.connect)
            }
        }
    }
}
