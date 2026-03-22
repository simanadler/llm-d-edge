package types

import "time"

// DeviceProfile represents the hardware capabilities of a device
type DeviceProfile struct {
	Platform     string           `json:"platform"`
	CPU          CPUInfo          `json:"cpu"`
	Memory       MemoryInfo       `json:"memory"`
	GPU          *GPUInfo         `json:"gpu,omitempty"`
	Storage      StorageInfo      `json:"storage"`
	Capabilities CapabilityScores `json:"capabilities"`
	Timestamp    time.Time        `json:"timestamp"`
}

// CPUInfo contains CPU details
type CPUInfo struct {
	Architecture string `json:"architecture"` // "arm64", "x86_64", etc.
	Model        string `json:"model"`
	Cores        int    `json:"cores"`
	ThreadsPerCore int  `json:"threads_per_core"`
	IsAppleSilicon bool `json:"is_apple_silicon"`
}

// MemoryInfo contains memory details
type MemoryInfo struct {
	TotalGB   float64 `json:"total_gb"`
	AvailableGB float64 `json:"available_gb"`
	Unified   bool    `json:"unified"` // True for Apple Silicon
	Type      string  `json:"type"`    // "DDR4", "DDR5", "LPDDR5", etc.
}

// GPUInfo contains GPU details
type GPUInfo struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`         // "integrated", "discrete"
	VRAMGB       float64 `json:"vram_gb"`
	ComputeUnits int     `json:"compute_units"` // Metal cores, CUDA cores, etc.
	Backend      string  `json:"backend"`       // "metal", "cuda", "opencl", etc.
}

// StorageInfo contains storage details
type StorageInfo struct {
	TotalGB     float64 `json:"total_gb"`
	AvailableGB float64 `json:"available_gb"`
	Type        string  `json:"type"` // "ssd", "hdd", "nvme"
}

// CapabilityScores represents device capability for different model sizes
type CapabilityScores struct {
	ModelSizeRanges map[string]CapabilityScore `json:"model_size_ranges"`
}

// CapabilityScore represents capability for a specific model size range
type CapabilityScore struct {
	Feasibility  float64 `json:"feasibility"`   // 0.0-1.0
	Performance  string  `json:"performance"`   // "excellent", "good", "acceptable", "poor", "infeasible"
	EstimatedTPS int     `json:"estimated_tps"` // Tokens per second
	MemoryFit    bool    `json:"memory_fit"`
}

// UserNeeds represents user requirements and preferences
type UserNeeds struct {
	Declared DeclaredPreferences `json:"declared"`
	Inferred InferredNeeds       `json:"inferred"`
	Combined CombinedScores      `json:"combined"`
}

// DeclaredPreferences are explicitly stated by the user
type DeclaredPreferences struct {
	PrimaryTasks       []string `json:"primary_tasks"`
	QualityPreference  string   `json:"quality_preference"`  // "low", "medium", "high", "premium"
	PrivacyRequirement string   `json:"privacy_requirement"` // "local_only", "local_preferred", "cloud_acceptable"
	StorageLimitGB     int      `json:"storage_limit_gb"`
	LatencyToleranceMS int      `json:"latency_tolerance_ms"`
}

// InferredNeeds are learned from usage patterns
type InferredNeeds struct {
	TaskDistribution   map[string]float64 `json:"task_distribution"`
	AvgPromptLength    int                `json:"avg_prompt_length"`
	QualityThreshold   float64            `json:"quality_threshold"`
	LatencyToleranceMS int                `json:"latency_tolerance_ms"`
	PreferredModels    []string           `json:"preferred_models"`
}

// CombinedScores merges declared and inferred needs
type CombinedScores struct {
	Tasks   map[string]float64 `json:"tasks"`   // Task importance scores
	Domains map[string]float64 `json:"domains"` // Domain importance scores
}

// ModelMetadata contains information about a model
type ModelMetadata struct {
	Name            string             `json:"name"`
	ParameterCount  string             `json:"parameter_count"` // "3B", "7B", "13B", etc.
	ContextLength   int                `json:"context_length"`
	ModelFamily     string             `json:"model_family"` // "llama", "mistral", "qwen", etc.
	QualityTier     string             `json:"quality_tier"` // "base", "instruct", "chat", etc.
	Tasks           map[string]float64 `json:"tasks"`        // Task capability scores
	Domains         map[string]float64 `json:"domains"`      // Domain expertise scores
	License         string             `json:"license"`
	DownloadSizeGB  float64            `json:"download_size_gb"`
	Formats         []string           `json:"formats"`         // "mlx", "gguf", "safetensors", etc.
	Quantizations   []string           `json:"quantizations"`   // "q4", "q8", "fp16", etc.
	HuggingFaceRepo string             `json:"huggingface_repo"`
}

// ModelCompatibility represents compatibility assessment
type ModelCompatibility struct {
	Model                   ModelMetadata `json:"model"`
	Compatible              bool          `json:"compatible"`
	Confidence              float64       `json:"confidence"`
	RecommendedQuantization string        `json:"recommended_quantization"`
	RecommendedFormat       string        `json:"recommended_format"`
	EstimatedMemoryGB       float64       `json:"estimated_memory_gb"`
	EstimatedTokensPerSec   int           `json:"estimated_tokens_per_sec"`
	Warnings                []string      `json:"warnings"`
	Optimizations           []string      `json:"optimizations"`
}

// ModelRecommendation represents a ranked model recommendation
type ModelRecommendation struct {
	Rank           int            `json:"rank"`
	Model          ModelMetadata  `json:"model"`
	Score          float64        `json:"score"`
	ScoreBreakdown ScoreBreakdown `json:"score_breakdown"`
	Explanation    Explanation    `json:"explanation"`
	Setup          SetupInfo      `json:"setup"`
}

// ScoreBreakdown shows how the score was calculated
type ScoreBreakdown struct {
	DeviceFit     float64 `json:"device_fit"`
	TaskAlignment float64 `json:"task_alignment"`
	Quality       float64 `json:"quality"`
	Efficiency    float64 `json:"efficiency"`
	Accessibility float64 `json:"accessibility"`
}

// Explanation provides human-readable reasoning
type Explanation struct {
	Summary      string   `json:"summary"`
	Strengths    []string `json:"strengths"`
	Tradeoffs    []string `json:"tradeoffs"`
	Alternatives []string `json:"alternatives"`
}

// SetupInfo provides installation details
type SetupInfo struct {
	InstallCommand string   `json:"install_command"`
	EstimatedTime  string   `json:"estimated_time"`
	Requirements   []string `json:"requirements"`
	PostInstall    []string `json:"post_install"`
}

// ScoringWeights defines the importance of different scoring criteria
type ScoringWeights struct {
	DeviceFit     float64 `json:"device_fit"`     // Default: 0.30
	TaskAlignment float64 `json:"task_alignment"` // Default: 0.35
	Quality       float64 `json:"quality"`        // Default: 0.20
	Efficiency    float64 `json:"efficiency"`     // Default: 0.10
	Accessibility float64 `json:"accessibility"`  // Default: 0.05
}

// DefaultScoringWeights returns the default scoring weights
func DefaultScoringWeights() ScoringWeights {
	return ScoringWeights{
		DeviceFit:     0.30,
		TaskAlignment: 0.35,
		Quality:       0.20,
		Efficiency:    0.10,
		Accessibility: 0.05,
	}
}

// HighEndScoringWeights returns weights optimized for high-end devices
// Prioritizes quality over efficiency
func HighEndScoringWeights() ScoringWeights {
	return ScoringWeights{
		DeviceFit:     0.25,
		TaskAlignment: 0.30,
		Quality:       0.35, // Increased from 0.20
		Efficiency:    0.05, // Decreased from 0.10
		Accessibility: 0.05,
	}
}

// AdaptiveScoringWeights returns weights based on device capabilities
func AdaptiveScoringWeights(memoryGB float64) ScoringWeights {
	// High-end device (32GB+): prioritize quality
	if memoryGB >= 32 {
		return HighEndScoringWeights()
	}
	
	// Mid-range device (16-32GB): balanced
	if memoryGB >= 16 {
		return ScoringWeights{
			DeviceFit:     0.28,
			TaskAlignment: 0.32,
			Quality:       0.25,
			Efficiency:    0.10,
			Accessibility: 0.05,
		}
	}
	
	// Low-end device (<16GB): prioritize efficiency
	return ScoringWeights{
		DeviceFit:     0.35,
		TaskAlignment: 0.30,
		Quality:       0.15,
		Efficiency:    0.15,
		Accessibility: 0.05,
	}
}

// Made with Bob
