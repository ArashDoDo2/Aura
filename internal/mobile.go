package internal

import (
	"context"
	"errors"
	"sync"
)

var (
	runningCtx    context.Context
	runningCancel context.CancelFunc
	runningMutex  sync.Mutex
)

// StartAuraClient starts the Aura SOCKS5 proxy for mobile/Android use.
// Returns error if startup fails. Call StopAuraClient() to stop gracefully.
func StartAuraClient(dnsServer string, domain string) error {
	runningMutex.Lock()
	if runningCtx != nil {
		runningMutex.Unlock()
		return errors.New("Aura client already running")
	}
	runningCtx, runningCancel = context.WithCancel(context.Background())
	runningMutex.Unlock()

	client := NewAuraClient(dnsServer)
	go func() {
		_ = client.StartSocks5(runningCtx, domain)
	}()
	return nil
}

// StopAuraClient stops the running Aura client gracefully.
func StopAuraClient() {
	runningMutex.Lock()
	if runningCancel != nil {
		runningCancel()
		runningCancel = nil
		runningCtx = nil
	}
	runningMutex.Unlock()
}
