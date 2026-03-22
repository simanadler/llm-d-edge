package platform

import (
	"context"
	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// DeviceProfiler is the interface that platform-specific implementations must satisfy
type DeviceProfiler interface {
	// Profile generates a complete device profile
	Profile(ctx context.Context) (*types.DeviceProfile, error)
	
	// DetectCPU detects CPU information
	DetectCPU(ctx context.Context) (*types.CPUInfo, error)
	
	// DetectMemory detects memory information
	DetectMemory(ctx context.Context) (*types.MemoryInfo, error)
	
	// DetectGPU detects GPU information (may return nil if no GPU)
	DetectGPU(ctx context.Context) (*types.GPUInfo, error)
	
	// DetectStorage detects storage information
	DetectStorage(ctx context.Context) (*types.StorageInfo, error)
	
	// CalculateCapabilities computes capability scores based on hardware
	CalculateCapabilities(ctx context.Context, profile *types.DeviceProfile) (types.CapabilityScores, error)
	
	// RunBenchmark runs a quick performance benchmark (optional, may be slow)
	RunBenchmark(ctx context.Context) (map[string]interface{}, error)
}

// SystemMonitor provides runtime system monitoring
type SystemMonitor interface {
	// GetBatteryLevel returns battery percentage (0-100) or -1 if not applicable
	GetBatteryLevel(ctx context.Context) (int, error)
	
	// GetThermalState returns thermal state ("nominal", "fair", "serious", "critical")
	GetThermalState(ctx context.Context) (string, error)
	
	// IsPluggedIn returns true if device is connected to power
	IsPluggedIn(ctx context.Context) (bool, error)
}

// Made with Bob
