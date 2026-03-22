package converter

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// MLXConverter converts HuggingFace models to MLX format (macOS only)
type MLXConverter struct{}

// NewMLXConverter creates a new MLX converter
func NewMLXConverter() *MLXConverter {
	return &MLXConverter{}
}

// Name returns the converter name
func (c *MLXConverter) Name() string {
	return "mlx"
}

// SupportsFormat checks if this converter supports the target format
func (c *MLXConverter) SupportsFormat(format string) bool {
	return strings.ToLower(format) == "mlx"
}

// IsAvailable checks if mlx_lm is installed
func (c *MLXConverter) IsAvailable() error {
	cmd := exec.Command("python3", "-c", "import mlx_lm")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mlx_lm not installed. Install with: pip install mlx-lm")
	}
	return nil
}

// Convert converts a HuggingFace model to MLX format
func (c *MLXConverter) Convert(ctx context.Context, model types.ModelMetadata, destPath string, opts ConvertOptions) error {
	// Check if mlx_lm is available
	if err := c.IsAvailable(); err != nil {
		return err
	}

	// Build conversion command
	args := []string{
		"-m", "mlx_lm.convert",
		"--hf-path", model.HuggingFaceRepo,
		"--mlx-path", destPath,
	}

	// Add quantization if specified
	if opts.Quantize {
		args = append(args, "--quantize")
		if opts.QuantizationBits > 0 {
			args = append(args, "--q-bits", fmt.Sprintf("%d", opts.QuantizationBits))
		} else {
			// Default to 4-bit quantization
			args = append(args, "--q-bits", "4")
		}
	}

	// Add optional parameters
	if opts.QuantizationGroup > 0 {
		args = append(args, "--q-group-size", fmt.Sprintf("%d", opts.QuantizationGroup))
	}

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(fmt.Sprintf("Converting %s to MLX format...", model.Name))
	}

	// Execute conversion
	cmd := exec.CommandContext(ctx, "python3", args...)
	
	// Capture output for progress reporting and error details
	var stdout, stderr strings.Builder
	if opts.ProgressCallback != nil {
		cmd.Stdout = &progressWriter{callback: opts.ProgressCallback, builder: &stdout}
		cmd.Stderr = &progressWriter{callback: opts.ProgressCallback, builder: &stderr}
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		if errMsg != "" {
			return fmt.Errorf("mlx conversion failed: %w\nOutput: %s", err, errMsg)
		}
		return fmt.Errorf("mlx conversion failed: %w", err)
	}

	if opts.ProgressCallback != nil {
		opts.ProgressCallback("MLX conversion complete")
	}

	return nil
}

// progressWriter captures command output and reports progress
type progressWriter struct {
	callback func(string)
	builder  *strings.Builder
}

func (w *progressWriter) Write(p []byte) (n int, err error) {
	// Always write to builder if present
	if w.builder != nil {
		w.builder.Write(p)
	}
	
	// Also send to callback if present
	if w.callback != nil {
		msg := strings.TrimSpace(string(p))
		if msg != "" {
			w.callback(msg)
		}
	}
	return len(p), nil
}

// Made with Bob