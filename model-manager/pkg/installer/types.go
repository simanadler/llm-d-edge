package installer

import (
	"time"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/downloader"
	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// InstallOptions contains options for installing a model
type InstallOptions struct {
	// Format specifies the model format (mlx, gguf, safetensors, etc.)
	Format string
	
	// Quantization specifies the quantization level (q4, q8, fp16, etc.)
	Quantization string
	
	// ProgressCallback is called with installation progress updates
	ProgressCallback func(progress downloader.DownloadProgress)
	
	// Force reinstalls even if already installed
	Force bool
}

// InstallResult contains the result of a model installation
type InstallResult struct {
	// Model is the installed model metadata
	Model types.ModelMetadata
	
	// InstallPath is the path where the model was installed
	InstallPath string
	
	// Format is the installed format
	Format string
	
	// Quantization is the installed quantization
	Quantization string
	
	// DownloaderUsed is the name of the downloader that was used
	DownloaderUsed string
}

// InstalledModel represents an installed model
type InstalledModel struct {
	// Model is the model metadata
	Model types.ModelMetadata
	
	// InstallPath is the installation path
	InstallPath string
	
	// InstalledAt is when the model was installed
	InstalledAt time.Time
}

// Made with Bob