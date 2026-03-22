package macos

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// MacOSProfiler implements platform.DeviceProfiler for macOS
type MacOSProfiler struct{}

// NewMacOSProfiler creates a new macOS device profiler
func NewMacOSProfiler() *MacOSProfiler {
	return &MacOSProfiler{}
}

// Profile generates a complete device profile
func (p *MacOSProfiler) Profile(ctx context.Context) (*types.DeviceProfile, error) {
	cpu, err := p.DetectCPU(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect CPU: %w", err)
	}

	memory, err := p.DetectMemory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect memory: %w", err)
	}

	gpu, err := p.DetectGPU(ctx)
	if err != nil {
		// GPU detection is optional, log but don't fail
		gpu = nil
	}

	storage, err := p.DetectStorage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect storage: %w", err)
	}

	profile := &types.DeviceProfile{
		Platform:  "darwin",
		CPU:       *cpu,
		Memory:    *memory,
		GPU:       gpu,
		Storage:   *storage,
		Timestamp: time.Now(),
	}

	// Calculate capabilities based on hardware
	capabilities, err := p.CalculateCapabilities(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate capabilities: %w", err)
	}
	profile.Capabilities = capabilities

	return profile, nil
}

// DetectCPU detects CPU information using sysctl
func (p *MacOSProfiler) DetectCPU(ctx context.Context) (*types.CPUInfo, error) {
	// Get CPU brand string
	brandCmd := exec.CommandContext(ctx, "sysctl", "-n", "machdep.cpu.brand_string")
	brandOutput, err := brandCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU brand: %w", err)
	}
	brand := strings.TrimSpace(string(brandOutput))

	// Get core count
	coresCmd := exec.CommandContext(ctx, "sysctl", "-n", "hw.physicalcpu")
	coresOutput, err := coresCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get core count: %w", err)
	}
	cores, err := strconv.Atoi(strings.TrimSpace(string(coresOutput)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse core count: %w", err)
	}

	// Get logical CPU count (threads)
	logicalCmd := exec.CommandContext(ctx, "sysctl", "-n", "hw.logicalcpu")
	logicalOutput, err := logicalCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get logical CPU count: %w", err)
	}
	logical, err := strconv.Atoi(strings.TrimSpace(string(logicalOutput)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse logical CPU count: %w", err)
	}

	threadsPerCore := 1
	if cores > 0 {
		threadsPerCore = logical / cores
	}

	// Detect architecture
	arch := runtime.GOARCH
	isAppleSilicon := arch == "arm64"

	return &types.CPUInfo{
		Architecture:   arch,
		Model:          brand,
		Cores:          cores,
		ThreadsPerCore: threadsPerCore,
		IsAppleSilicon: isAppleSilicon,
	}, nil
}

// DetectMemory detects memory information
func (p *MacOSProfiler) DetectMemory(ctx context.Context) (*types.MemoryInfo, error) {
	// Get total memory
	memCmd := exec.CommandContext(ctx, "sysctl", "-n", "hw.memsize")
	memOutput, err := memCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory size: %w", err)
	}
	memBytes, err := strconv.ParseInt(strings.TrimSpace(string(memOutput)), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory size: %w", err)
	}
	totalGB := float64(memBytes) / (1024 * 1024 * 1024)

	// Get available memory using vm_stat
	vmCmd := exec.CommandContext(ctx, "vm_stat")
	vmOutput, err := vmCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get vm_stat: %w", err)
	}

	// Parse vm_stat output to calculate available memory
	availableGB := p.parseAvailableMemory(string(vmOutput), totalGB)

	// Detect if unified memory (Apple Silicon)
	arch := runtime.GOARCH
	isUnified := arch == "arm64"

	memType := "DDR4"
	if isUnified {
		memType = "LPDDR5" // Apple Silicon uses LPDDR5
	}

	return &types.MemoryInfo{
		TotalGB:     totalGB,
		AvailableGB: availableGB,
		Unified:     isUnified,
		Type:        memType,
	}, nil
}

// parseAvailableMemory parses vm_stat output to estimate available memory
func (p *MacOSProfiler) parseAvailableMemory(vmStat string, totalGB float64) float64 {
	// This is a simplified calculation
	// In production, you'd parse free, inactive, and speculative pages
	// For now, estimate 70% of total as available (conservative)
	return totalGB * 0.7
}

// DetectGPU detects GPU information
func (p *MacOSProfiler) DetectGPU(ctx context.Context) (*types.GPUInfo, error) {
	// Use system_profiler to get GPU info
	cmd := exec.CommandContext(ctx, "system_profiler", "SPDisplaysDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get GPU info: %w", err)
	}

	// For now, return basic info
	// In production, parse the JSON output properly
	arch := runtime.GOARCH
	if arch == "arm64" {
		// Apple Silicon - integrated GPU
		return &types.GPUInfo{
			Name:         "Apple GPU",
			Type:         "integrated",
			VRAMGB:       0, // Unified memory
			ComputeUnits: 0, // Would need to detect M1/M2/M3 variant
			Backend:      "metal",
		}, nil
	}

	// Intel Mac - might have discrete GPU
	// Parse system_profiler output to detect
	gpuInfo := p.parseGPUInfo(string(output))
	return gpuInfo, nil
}

// parseGPUInfo parses system_profiler output for GPU details
func (p *MacOSProfiler) parseGPUInfo(output string) *types.GPUInfo {
	// Simplified parsing - in production, parse JSON properly
	return &types.GPUInfo{
		Name:         "Unknown GPU",
		Type:         "integrated",
		VRAMGB:       0,
		ComputeUnits: 0,
		Backend:      "metal",
	}
}

// DetectStorage detects storage information
func (p *MacOSProfiler) DetectStorage(ctx context.Context) (*types.StorageInfo, error) {
	// Use df to get storage info for root volume
	cmd := exec.CommandContext(ctx, "df", "-k", "/")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get storage info: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected df output")
	}

	// Parse the second line (first line is headers)
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return nil, fmt.Errorf("unexpected df output format")
	}

	// Fields: Filesystem, 1K-blocks, Used, Available, Capacity, Mounted
	totalKB, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse total storage: %w", err)
	}
	availableKB, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse available storage: %w", err)
	}

	totalGB := float64(totalKB) / (1024 * 1024)
	availableGB := float64(availableKB) / (1024 * 1024)

	// Assume SSD for modern Macs
	storageType := "ssd"

	return &types.StorageInfo{
		TotalGB:     totalGB,
		AvailableGB: availableGB,
		Type:        storageType,
	}, nil
}

// CalculateCapabilities computes capability scores based on hardware
func (p *MacOSProfiler) CalculateCapabilities(ctx context.Context, profile *types.DeviceProfile) (types.CapabilityScores, error) {
	scores := types.CapabilityScores{
		ModelSizeRanges: make(map[string]types.CapabilityScore),
	}

	// Define model size ranges and calculate scores
	ranges := []struct {
		name       string
		minMemoryGB float64
		minCores   int
	}{
		{"0.5B-1B", 2, 2},
		{"1B-3B", 4, 4},
		{"3B-7B", 8, 4},
		{"7B-13B", 16, 6},
		{"13B-30B", 32, 8},
		{"30B+", 64, 12},
	}

	for _, r := range ranges {
		score := p.calculateRangeScore(profile, r.minMemoryGB, r.minCores)
		scores.ModelSizeRanges[r.name] = score
	}

	return scores, nil
}

// calculateRangeScore calculates capability score for a model size range
func (p *MacOSProfiler) calculateRangeScore(profile *types.DeviceProfile, minMemoryGB float64, minCores int) types.CapabilityScore {
	memoryFit := profile.Memory.TotalGB >= minMemoryGB
	coresFit := profile.CPU.Cores >= minCores

	var feasibility float64
	var performance string
	var estimatedTPS int

	if !memoryFit {
		feasibility = 0.0
		performance = "infeasible"
		estimatedTPS = 0
	} else if !coresFit {
		feasibility = 0.5
		performance = "poor"
		estimatedTPS = 5
	} else {
		// Calculate based on how much headroom we have
		memoryRatio := profile.Memory.TotalGB / minMemoryGB
		coresRatio := float64(profile.CPU.Cores) / float64(minCores)
		
		avgRatio := (memoryRatio + coresRatio) / 2
		
		if avgRatio >= 2.0 {
			feasibility = 1.0
			performance = "excellent"
			estimatedTPS = 50
		} else if avgRatio >= 1.5 {
			feasibility = 0.9
			performance = "good"
			estimatedTPS = 30
		} else {
			feasibility = 0.7
			performance = "acceptable"
			estimatedTPS = 15
		}
		
		// Boost for Apple Silicon
		if profile.CPU.IsAppleSilicon {
			estimatedTPS = int(float64(estimatedTPS) * 1.5)
		}
	}

	return types.CapabilityScore{
		Feasibility:  feasibility,
		Performance:  performance,
		EstimatedTPS: estimatedTPS,
		MemoryFit:    memoryFit,
	}
}

// RunBenchmark runs a quick performance benchmark
func (p *MacOSProfiler) RunBenchmark(ctx context.Context) (map[string]interface{}, error) {
	// Placeholder for actual benchmark implementation
	// In production, this would run actual inference tests
	return map[string]interface{}{
		"status": "not_implemented",
		"note":   "Benchmark functionality coming soon",
	}, nil
}

// Made with Bob
