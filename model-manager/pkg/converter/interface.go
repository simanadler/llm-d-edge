package converter

import (
	"context"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// ModelConverter converts models to platform-specific formats
type ModelConverter interface {
	// Convert converts a model to the target format
	Convert(ctx context.Context, model types.ModelMetadata, destPath string, opts ConvertOptions) error
	
	// SupportsFormat checks if this converter supports the target format
	SupportsFormat(format string) bool
	
	// IsAvailable checks if the converter tools are installed
	IsAvailable() error
	
	// Name returns the converter name
	Name() string
}

// ConvertOptions contains options for model conversion
type ConvertOptions struct {
	// TargetFormat is the desired output format (mlx, gguf, etc.)
	TargetFormat string
	
	// Quantize enables quantization during conversion
	Quantize bool
	
	// QuantizationBits specifies the quantization level (4, 8, etc.)
	QuantizationBits int
	
	// QuantizationGroup specifies the quantization group size
	QuantizationGroup int
	
	// ProgressCallback is called with progress updates
	ProgressCallback func(message string)
}

// Made with Bob