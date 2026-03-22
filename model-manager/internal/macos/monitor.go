package macos

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// MacOSMonitor implements platform.SystemMonitor for macOS
type MacOSMonitor struct{}

// NewMacOSMonitor creates a new macOS system monitor
func NewMacOSMonitor() *MacOSMonitor {
	return &MacOSMonitor{}
}

// GetBatteryLevel returns battery percentage (0-100) or -1 if not applicable
func (m *MacOSMonitor) GetBatteryLevel(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "pmset", "-g", "batt")
	output, err := cmd.Output()
	if err != nil {
		return -1, fmt.Errorf("failed to get battery info: %w", err)
	}

	// Parse output like: "Now drawing from 'Battery Power'\n -InternalBattery-0 (id=1234567)	95%; discharging; 5:23 remaining present: true"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "InternalBattery") {
			// Extract percentage
			parts := strings.Split(line, "\t")
			if len(parts) >= 2 {
				percentStr := strings.TrimSpace(strings.Split(parts[1], ";")[0])
				percentStr = strings.TrimSuffix(percentStr, "%")
				percent, err := strconv.Atoi(percentStr)
				if err != nil {
					return -1, fmt.Errorf("failed to parse battery percentage: %w", err)
				}
				return percent, nil
			}
		}
	}

	// No battery found (desktop Mac)
	return -1, nil
}

// GetThermalState returns thermal state
func (m *MacOSMonitor) GetThermalState(ctx context.Context) (string, error) {
	// Use powermetrics to get thermal pressure (requires sudo, so fallback to nominal)
	// In production, you might use IOKit APIs via CGO
	
	// For now, return nominal as we can't easily get thermal state without sudo
	// A proper implementation would use:
	// - IOKit's thermal notification APIs
	// - Or parse /usr/bin/powermetrics output (requires sudo)
	return "nominal", nil
}

// IsPluggedIn returns true if device is connected to power
func (m *MacOSMonitor) IsPluggedIn(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "pmset", "-g", "batt")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get power info: %w", err)
	}

	// Check if drawing from AC Power
	outputStr := string(output)
	if strings.Contains(outputStr, "'AC Power'") {
		return true, nil
	}
	if strings.Contains(outputStr, "'Battery Power'") {
		return false, nil
	}

	// Desktop Macs are always plugged in
	return true, nil
}

// Made with Bob
