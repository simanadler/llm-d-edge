// +build darwin

package platform

import (
	"github.com/simanadler/llm-d-edge/model-manager/internal/macos"
)

// newMacOSProfiler creates a macOS device profiler
func newMacOSProfiler() (DeviceProfiler, error) {
	return macos.NewMacOSProfiler(), nil
}

// newMacOSMonitor creates a macOS system monitor
func newMacOSMonitor() (SystemMonitor, error) {
	return macos.NewMacOSMonitor(), nil
}

// Made with Bob
