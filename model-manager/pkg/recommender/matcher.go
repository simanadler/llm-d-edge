package recommender

import (
	"context"
	"fmt"
	"strings"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// ModelMatcher discovers and analyzes model compatibility
type ModelMatcher struct {
	registry ModelRegistry
}

// NewModelMatcher creates a new model matcher with default registries
func NewModelMatcher() *ModelMatcher {
	// Create composite registry with HuggingFace as default
	registry := NewCompositeRegistry(
		NewHuggingFaceRegistry(),
	)
	
	return &ModelMatcher{
		registry: registry,
	}
}

// NewModelMatcherWithRegistry creates a matcher with a custom registry
func NewModelMatcherWithRegistry(registry ModelRegistry) *ModelMatcher {
	return &ModelMatcher{
		registry: registry,
	}
}

// FindCandidates discovers available models that might work on the device
func (m *ModelMatcher) FindCandidates(ctx context.Context, device types.DeviceProfile) ([]types.ModelCompatibility, error) {
	// Get models from registry
	models, err := m.registry.GetModels(ctx)
	if err != nil {
		// Fallback to curated list if registry fails
		models = m.getCuratedModels()
	}
	
	compatibilities := make([]types.ModelCompatibility, 0, len(models))

	for _, model := range models {
		compat, err := m.CheckCompatibility(ctx, model, device)
		if err != nil {
			// Log error but continue with other models
			continue
		}
		compatibilities = append(compatibilities, *compat)
	}

	return compatibilities, nil
}

// CheckCompatibility assesses if a model is compatible with the device
func (m *ModelMatcher) CheckCompatibility(
	ctx context.Context,
	model types.ModelMetadata,
	device types.DeviceProfile,
) (*types.ModelCompatibility, error) {
	compat := &types.ModelCompatibility{
		Model:      model,
		Compatible: false,
		Warnings:   []string{},
		Optimizations: []string{},
	}

	// Estimate memory requirements based on parameter count
	memoryGB := m.estimateMemoryRequirement(model.ParameterCount)
	compat.EstimatedMemoryGB = memoryGB

	// Check if model fits in memory
	if memoryGB > device.Memory.TotalGB {
		compat.Compatible = false
		compat.Confidence = 0.0
		compat.Warnings = append(compat.Warnings, 
			fmt.Sprintf("Model requires %.1f GB but device has %.1f GB", memoryGB, device.Memory.TotalGB))
		return compat, nil
	}

	// Model fits, mark as compatible
	compat.Compatible = true

	// Calculate confidence based on memory headroom
	memoryRatio := memoryGB / device.Memory.TotalGB
	if memoryRatio < 0.5 {
		compat.Confidence = 0.95
	} else if memoryRatio < 0.7 {
		compat.Confidence = 0.85
	} else if memoryRatio < 0.9 {
		compat.Confidence = 0.70
		compat.Warnings = append(compat.Warnings, "Model will use significant memory")
	} else {
		compat.Confidence = 0.50
		compat.Warnings = append(compat.Warnings, "Model will use most available memory")
	}

	// Recommend format based on platform
	compat.RecommendedFormat = m.recommendFormat(device)
	
	// Recommend quantization based on memory constraints
	compat.RecommendedQuantization = m.recommendQuantization(memoryRatio)

	// Estimate performance
	compat.EstimatedTokensPerSec = m.estimatePerformance(model, device)

	// Add platform-specific optimizations
	if device.CPU.IsAppleSilicon {
		compat.Optimizations = append(compat.Optimizations, 
			"Use MLX format for optimal Apple Silicon performance")
	}

	return compat, nil
}

// estimateMemoryRequirement estimates memory needed for a model
func (m *ModelMatcher) estimateMemoryRequirement(paramCount string) float64 {
	// Rough estimates for different model sizes (in GB)
	// These assume quantized models (Q4 or Q8)
	estimates := map[string]float64{
		"0.5B": 1.0,
		"1B":   2.0,
		"3B":   4.0,
		"7B":   8.0,
		"13B":  16.0,
		"30B":  32.0,
		"70B":  64.0,
	}

	// Try exact match first
	if mem, ok := estimates[paramCount]; ok {
		return mem
	}

	// Try to parse and estimate
	paramCount = strings.ToUpper(paramCount)
	if strings.HasSuffix(paramCount, "B") {
		// Default to 2GB per billion parameters (conservative)
		return 2.0
	}

	return 4.0 // Default fallback
}

// recommendFormat recommends the best model format for the platform
func (m *ModelMatcher) recommendFormat(device types.DeviceProfile) string {
	if device.CPU.IsAppleSilicon {
		return "mlx" // MLX is optimal for Apple Silicon
	}
	return "gguf" // GGUF is widely compatible
}

// recommendQuantization recommends quantization level based on memory constraints
func (m *ModelMatcher) recommendQuantization(memoryRatio float64) string {
	if memoryRatio < 0.4 {
		return "q8" // Plenty of memory, use higher quality
	} else if memoryRatio < 0.7 {
		return "q4" // Balanced
	} else {
		return "q4" // Tight on memory, use aggressive quantization
	}
}

// estimatePerformance estimates tokens per second
func (m *ModelMatcher) estimatePerformance(model types.ModelMetadata, device types.DeviceProfile) int {
	// Base performance on CPU cores and architecture
	basePerf := device.CPU.Cores * 5 // 5 tokens/sec per core baseline

	// Boost for Apple Silicon
	if device.CPU.IsAppleSilicon {
		basePerf = int(float64(basePerf) * 2.0)
	}

	// Adjust for model size (larger models are slower)
	sizeMultiplier := 1.0
	switch model.ParameterCount {
	case "0.5B", "1B":
		sizeMultiplier = 1.5
	case "3B":
		sizeMultiplier = 1.0
	case "7B":
		sizeMultiplier = 0.7
	case "13B":
		sizeMultiplier = 0.4
	case "30B", "70B":
		sizeMultiplier = 0.2
	}

	return int(float64(basePerf) * sizeMultiplier)
}

// getCuratedModels returns a curated list of popular models
// In production, this would query from HuggingFace or a model registry
func (m *ModelMatcher) getCuratedModels() []types.ModelMetadata {
	return []types.ModelMetadata{
		{
			Name:            "Qwen2.5-0.5B-Instruct",
			ParameterCount:  "0.5B",
			ContextLength:   32768,
			ModelFamily:     "qwen",
			QualityTier:     "instruct",
			Tasks:           map[string]float64{"general": 0.7, "code": 0.6, "chat": 0.8},
			Domains:         map[string]float64{"general": 0.8},
			License:         "apache-2.0",
			DownloadSizeGB:  0.5,
			Formats:         []string{"mlx", "gguf"},
			Quantizations:   []string{"q4", "q8", "fp16"},
			HuggingFaceRepo: "Qwen/Qwen2.5-0.5B-Instruct",
		},
		{
			Name:            "Qwen2.5-3B-Instruct",
			ParameterCount:  "3B",
			ContextLength:   32768,
			ModelFamily:     "qwen",
			QualityTier:     "instruct",
			Tasks:           map[string]float64{"general": 0.85, "code": 0.8, "chat": 0.9},
			Domains:         map[string]float64{"general": 0.9, "technical": 0.85},
			License:         "apache-2.0",
			DownloadSizeGB:  2.0,
			Formats:         []string{"mlx", "gguf"},
			Quantizations:   []string{"q4", "q8", "fp16"},
			HuggingFaceRepo: "Qwen/Qwen2.5-3B-Instruct",
		},
		{
			Name:            "Llama-3.2-3B-Instruct",
			ParameterCount:  "3B",
			ContextLength:   8192,
			ModelFamily:     "llama",
			QualityTier:     "instruct",
			Tasks:           map[string]float64{"general": 0.85, "chat": 0.9, "reasoning": 0.8},
			Domains:         map[string]float64{"general": 0.9},
			License:         "llama3.2",
			DownloadSizeGB:  2.0,
			Formats:         []string{"mlx", "gguf"},
			Quantizations:   []string{"q4", "q8", "fp16"},
			HuggingFaceRepo: "meta-llama/Llama-3.2-3B-Instruct",
		},
		{
			Name:            "Mistral-7B-Instruct-v0.3",
			ParameterCount:  "7B",
			ContextLength:   32768,
			ModelFamily:     "mistral",
			QualityTier:     "instruct",
			Tasks:           map[string]float64{"general": 0.9, "code": 0.85, "chat": 0.9},
			Domains:         map[string]float64{"general": 0.95, "technical": 0.9},
			License:         "apache-2.0",
			DownloadSizeGB:  4.5,
			Formats:         []string{"mlx", "gguf"},
			Quantizations:   []string{"q4", "q8"},
			HuggingFaceRepo: "mistralai/Mistral-7B-Instruct-v0.3",
		},
		{
			Name:            "Phi-3-mini-4k-instruct",
			ParameterCount:  "3B",
			ContextLength:   4096,
			ModelFamily:     "phi",
			QualityTier:     "instruct",
			Tasks:           map[string]float64{"general": 0.8, "reasoning": 0.85, "chat": 0.8},
			Domains:         map[string]float64{"general": 0.85, "education": 0.9},
			License:         "mit",
			DownloadSizeGB:  2.0,
			Formats:         []string{"gguf"},
			Quantizations:   []string{"q4", "q8"},
			HuggingFaceRepo: "microsoft/Phi-3-mini-4k-instruct",
		},
	}
}

// Made with Bob
