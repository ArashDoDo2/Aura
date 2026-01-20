package internal

import (
	"context"
	"fmt"
	"log"
)

var (
	globalClient *AuraClient
	globalCancel context.CancelFunc
	isRunning    bool
)

// StartTunnel starts the Aura DNS tunnel with SOCKS5 proxy
// This is the gomobile-compatible entry point for Flutter/Android
// dnsServer: DNS server address (empty = use system resolver)
// domain: Target domain (e.g., "example.com.")
// Returns empty string on success, error message on failure
func StartTunnel(dnsServer, domain string) string {
	if isRunning {
		return "Tunnel already running"
	}

	if domain == "" {
		return "Domain cannot be empty"
	}

	// Stop any existing instance
	if globalClient != nil {
		StopTunnel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	globalCancel = cancel

	// Use default port 1080 for SOCKS5
	globalClient = NewAuraClient(dnsServer, domain, 1080)

	go func() {
		err := globalClient.StartSocks5(ctx)
		if err != nil {
			log.Printf("Aura tunnel error: %v", err)
			isRunning = false
		}
	}()

	isRunning = true
	log.Printf("Aura tunnel started - DNS: %s, Domain: %s", dnsServer, domain)
	return ""
}

// StopTunnel gracefully stops the tunnel
// Returns empty string on success, error message on failure
func StopTunnel() string {
	if !isRunning {
		return "Tunnel not running"
	}

	if globalCancel != nil {
		globalCancel()
		globalCancel = nil
	}

	globalClient = nil
	isRunning = false
	log.Printf("Aura tunnel stopped")
	return ""
}

// IsRunning returns whether the tunnel is currently active
func IsRunning() bool {
	return isRunning
}

// GetStatus returns the current tunnel status as a string
func GetStatus() string {
	if !isRunning {
		return "stopped"
	}

	if globalClient == nil {
		return "error"
	}

	effectiveDNS := globalClient.DNSServer
	if effectiveDNS == "" {
		effectiveDNS = "system"
	}

	return fmt.Sprintf("running|dns=%s|domain=%s|port=%d|session=%s",
		effectiveDNS, globalClient.Domain, globalClient.Socks5Port, globalClient.SessionID)
}

// Legacy functions for backward compatibility
func StartAuraClient(dnsServer, domain string, port int) error {
	if port == 0 {
		port = 1080
	}

	if globalClient != nil {
		StopAuraClient()
	}

	ctx, cancel := context.WithCancel(context.Background())
	globalCancel = cancel

	globalClient = NewAuraClient(dnsServer, domain, port)
	go globalClient.StartSocks5(ctx)
	isRunning = true
	return nil
}

func StopAuraClient() {
	if globalCancel != nil {
		globalCancel()
	}
	globalClient = nil
	isRunning = false
}
