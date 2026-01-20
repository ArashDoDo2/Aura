package internal

import (
"context"
)

var (
globalClient *AuraClient
globalCancel context.CancelFunc
)

// StartAuraClient starts the Aura SOCKS5 proxy
// dnsServer example: "1.1.1.1:53" (empty = use system resolver)
// domain example: "example.com."
// port: SOCKS5 listen port (0 for default 1080)
func StartAuraClient(dnsServer, domain string, port int) error {
	if globalClient != nil {
		StopAuraClient()
	}

	ctx, cancel := context.WithCancel(context.Background())
	globalCancel = cancel

	globalClient = NewAuraClient(dnsServer, domain, port)
	go globalClient.StartSocks5(ctx)
	return nil
}

// StopAuraClient gracefully stops the proxy
func StopAuraClient() {
if globalCancel != nil {
globalCancel()
}
globalClient = nil
}
