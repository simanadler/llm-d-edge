package router

import (
	"net"
	"sync"
	"time"
)

// NetworkStatus tracks network connectivity
type NetworkStatus struct {
	mu          sync.RWMutex
	connected   bool
	lastCheck   time.Time
	checkTicker *time.Ticker
	stopChan    chan struct{}
}

// NewNetworkStatus creates a new network status tracker
func NewNetworkStatus() *NetworkStatus {
	ns := &NetworkStatus{
		connected:   true, // Assume connected initially
		lastCheck:   time.Now(),
		checkTicker: time.NewTicker(5 * time.Second),
		stopChan:    make(chan struct{}),
	}

	// Start background connectivity checker
	go ns.monitorConnectivity()

	return ns
}

// IsConnected returns whether network is currently connected
func (ns *NetworkStatus) IsConnected() bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.connected
}

// monitorConnectivity periodically checks network connectivity
func (ns *NetworkStatus) monitorConnectivity() {
	for {
		select {
		case <-ns.checkTicker.C:
			connected := ns.checkConnectivity()
			ns.mu.Lock()
			ns.connected = connected
			ns.lastCheck = time.Now()
			ns.mu.Unlock()

		case <-ns.stopChan:
			return
		}
	}
}

// checkConnectivity performs an actual connectivity check
func (ns *NetworkStatus) checkConnectivity() bool {
	// Try to resolve a well-known domain
	_, err := net.LookupHost("www.google.com")
	return err == nil
}

// Stop stops the network monitor
func (ns *NetworkStatus) Stop() {
	ns.checkTicker.Stop()
	close(ns.stopChan)
}

// Made with Bob
