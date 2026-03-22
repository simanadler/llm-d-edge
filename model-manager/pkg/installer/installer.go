package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/converter"
	"github.com/simanadler/llm-d-edge/model-manager/pkg/downloader"
	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// ModelInstaller handles model installation
type ModelInstaller struct {
	downloaders   []downloader.ModelDownloader
	converters    []converter.ModelConverter
	modelsDir     string
	deviceProfile *types.DeviceProfile
}

// NewModelInstaller creates a new model installer
func NewModelInstaller(modelsDir string) *ModelInstaller {
	if modelsDir == "" {
		// Default to platform-specific directory to match edge-router
		home, _ := os.UserHomeDir()
		// macOS: ~/Library/Application Support/llm-d/models
		// Windows: %APPDATA%\llm-d\models
		// Linux: ~/.local/share/llm-d/models
		modelsDir = getDefaultModelsDir(home)
	}

	return &ModelInstaller{
		downloaders: []downloader.ModelDownloader{
			downloader.NewHuggingFaceDownloader(),
			downloader.NewOllamaDownloader(),
		},
		converters: []converter.ModelConverter{},
		modelsDir:  modelsDir,
	}
}

// AddConverter adds a model converter to the installer
func (i *ModelInstaller) AddConverter(conv converter.ModelConverter) {
	i.converters = append(i.converters, conv)
}

// SetDeviceProfile sets the device profile for optimal quantization selection
func (i *ModelInstaller) SetDeviceProfile(profile *types.DeviceProfile) {
	i.deviceProfile = profile
}

// getDefaultModelsDir returns the platform-specific default models directory
func getDefaultModelsDir(home string) string {
	// Check if we're on macOS by looking for Library directory
	if _, err := os.Stat(filepath.Join(home, "Library")); err == nil {
		return filepath.Join(home, "Library", "Application Support", "llm-d", "models")
	}
	
	// Check if we're on Windows by looking for AppData
	if _, err := os.Stat(filepath.Join(home, "AppData")); err == nil {
		return filepath.Join(home, "AppData", "Roaming", "llm-d", "models")
	}
	
	// Default to Linux/Unix
	return filepath.Join(home, ".local", "share", "llm-d", "models")
}

// InstallModel installs a model
func (i *ModelInstaller) Install(ctx context.Context, model types.ModelMetadata, opts InstallOptions) (*InstallResult, error) {
	// Determine optimal quantization if not specified
	if opts.Quantization == "" {
		opts.Quantization = i.determineOptimalQuantization(model)
	}

	// Create model directory
	modelDir := filepath.Join(i.modelsDir, sanitizeModelName(model.Name))
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create model directory: %w", err)
	}

	// Check if we have a converter for the requested format
	var modelConverter converter.ModelConverter
	for _, c := range i.converters {
		if c.SupportsFormat(opts.Format) && c.IsAvailable() == nil {
			modelConverter = c
			break
		}
	}

	// If we have a converter, use it
	if modelConverter != nil {
		return i.installWithConversion(ctx, model, modelDir, opts, modelConverter)
	}

	// Otherwise, try direct download
	return i.installWithDownload(ctx, model, modelDir, opts)
}

// installWithConversion converts a model using the converter
func (i *ModelInstaller) installWithConversion(ctx context.Context, model types.ModelMetadata, modelDir string, opts InstallOptions, modelConverter converter.ModelConverter) (*InstallResult, error) {
	// Check if destination already exists
	if _, err := os.Stat(modelDir); err == nil {
		// Directory exists, remove it to allow fresh conversion
		if opts.ProgressCallback != nil {
			opts.ProgressCallback(downloader.DownloadProgress{
				Status: fmt.Sprintf("Removing existing installation at %s...", modelDir),
			})
		}
		if err := os.RemoveAll(modelDir); err != nil {
			return nil, fmt.Errorf("failed to remove existing installation: %w", err)
		}
	}

	// Wrap the progress callback to convert between types
	var convertProgressCallback func(string)
	if opts.ProgressCallback != nil {
		convertProgressCallback = func(msg string) {
			opts.ProgressCallback(downloader.DownloadProgress{
				Status: msg,
			})
		}
		convertProgressCallback(fmt.Sprintf("Converting %s to %s format...", model.Name, opts.Format))
	}

	// Prepare conversion options
	convertOpts := converter.ConvertOptions{
		TargetFormat:      opts.Format,
		Quantize:          true,
		QuantizationBits:  i.getQuantizationBits(opts.Quantization),
		QuantizationGroup: 64, // Default group size
		ProgressCallback:  convertProgressCallback,
	}

	// Convert the model
	if err := modelConverter.Convert(ctx, model, modelDir, convertOpts); err != nil {
		return nil, fmt.Errorf("failed to convert model: %w", err)
	}

	// Create installation result
	result := &InstallResult{
		Model:          model,
		InstallPath:    modelDir,
		Format:         opts.Format,
		Quantization:   opts.Quantization,
		DownloaderUsed: modelConverter.Name(),
	}

	// Save metadata
	if err := i.saveMetadata(model, modelDir, opts); err != nil {
		return result, fmt.Errorf("warning: failed to save metadata: %w", err)
	}

	return result, nil
}

// installWithDownload downloads a pre-converted model
func (i *ModelInstaller) installWithDownload(ctx context.Context, model types.ModelMetadata, modelDir string, opts InstallOptions) (*InstallResult, error) {
	// Find a suitable downloader
	var selectedDownloader downloader.ModelDownloader
	for _, d := range i.downloaders {
		if d.SupportsModel(model) {
			selectedDownloader = d
			break
		}
	}

	if selectedDownloader == nil {
		return nil, fmt.Errorf("no downloader found for model %s", model.Name)
	}

	// Prepare download options
	downloadOpts := downloader.DownloadOptions{
		Format:           opts.Format,
		Quantization:     opts.Quantization,
		ProgressCallback: opts.ProgressCallback,
		Resume:           true,
		Verify:           true,
	}

	// Download the model
	if err := selectedDownloader.Download(ctx, model, modelDir, downloadOpts); err != nil {
		return nil, fmt.Errorf("failed to download model: %w", err)
	}

	// Create installation result
	result := &InstallResult{
		Model:          model,
		InstallPath:    modelDir,
		Format:         opts.Format,
		Quantization:   opts.Quantization,
		DownloaderUsed: selectedDownloader.Name(),
	}

	// Save metadata
	if err := i.saveMetadata(model, modelDir, opts); err != nil {
		return result, fmt.Errorf("warning: failed to save metadata: %w", err)
	}

	return result, nil
}

// determineOptimalQuantization determines the best quantization based on device capabilities
func (i *ModelInstaller) determineOptimalQuantization(model types.ModelMetadata) string {
	if i.deviceProfile == nil {
		return "q4" // Default to 4-bit
	}

	// Determine based on available memory
	availableMemory := i.deviceProfile.Memory.AvailableGB
	
	// Estimate model memory requirements
	paramSize := estimateModelSize(model.ParameterCount)
	
	// If we have plenty of memory (4x model Q4 size), use Q8
	if availableMemory >= paramSize*2 {
		return "q8"
	}
	
	// Otherwise use Q4
	return "q4"
}

// getQuantizationBits converts quantization string to bits
func (i *ModelInstaller) getQuantizationBits(quant string) int {
	switch strings.ToLower(quant) {
	case "q8", "8bit":
		return 8
	case "q4", "4bit":
		return 4
	default:
		return 4
	}
}

// estimateModelSize estimates model size in GB based on parameter count (Q4)
func estimateModelSize(paramCount string) float64 {
	paramCount = strings.ToUpper(strings.TrimSpace(paramCount))
	paramCount = strings.TrimSuffix(paramCount, "B")
	
	sizes := map[string]float64{
		"0.5": 0.5,
		"1":   1.0,
		"1.7": 1.5,
		"3":   2.0,
		"7":   4.5,
		"8":   5.0,
		"13":  8.0,
		"14":  9.0,
		"30":  18.0,
		"32":  20.0,
		"70":  40.0,
		"72":  45.0,
	}
	
	if size, ok := sizes[paramCount]; ok {
		return size
	}
	
	return 5.0 // Default estimate
}

// ListInstalled returns a list of installed models
func (i *ModelInstaller) ListInstalled() ([]InstalledModel, error) {
	entries, err := os.ReadDir(i.modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []InstalledModel{}, nil
		}
		return nil, fmt.Errorf("failed to read models directory: %w", err)
	}

	installed := make([]InstalledModel, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		modelDir := filepath.Join(i.modelsDir, entry.Name())
		info, _ := entry.Info()
		
		// Try to load metadata first
		metadata, err := i.loadMetadata(modelDir)
		if err == nil {
			// Model installed by model-manager
			installed = append(installed, InstalledModel{
				Model:        *metadata,
				InstallPath:  modelDir,
				InstalledAt:  info.ModTime(),
			})
			continue
		}

		// Check if this is a manually installed model by looking for model files
		if i.isModelDirectory(modelDir) {
			// Create basic metadata for manually installed model
			installed = append(installed, InstalledModel{
				Model: types.ModelMetadata{
					Name:           entry.Name(),
					ParameterCount: "unknown",
					ModelFamily:    "unknown",
					QualityTier:    "unknown",
				},
				InstallPath:  modelDir,
				InstalledAt:  info.ModTime(),
			})
		}
	}

	return installed, nil
}

// isModelDirectory checks if a directory contains model files
func (i *ModelInstaller) isModelDirectory(dir string) bool {
	// Check for common model file patterns
	patterns := []string{
		"*.safetensors",
		"*.gguf",
		"*.bin",
		"model.safetensors",
		"pytorch_model.bin",
		"config.json",
		"weights.npz", // MLX format
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err == nil && len(matches) > 0 {
			return true
		}
	}

	return false
}

// Uninstall removes an installed model
func (i *ModelInstaller) Uninstall(modelName string) error {
	modelDir := filepath.Join(i.modelsDir, sanitizeModelName(modelName))
	
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return fmt.Errorf("model %s is not installed", modelName)
	}

	return os.RemoveAll(modelDir)
}

// GetModelPath returns the installation path for a model
func (i *ModelInstaller) GetModelPath(modelName string) (string, error) {
	modelDir := filepath.Join(i.modelsDir, sanitizeModelName(modelName))
	
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		return "", fmt.Errorf("model %s is not installed", modelName)
	}

	return modelDir, nil
}

// saveMetadata saves model metadata to the installation directory
func (i *ModelInstaller) saveMetadata(model types.ModelMetadata, modelDir string, opts InstallOptions) error {
	// Implementation would save metadata as JSON
	// For now, just create a marker file
	markerPath := filepath.Join(modelDir, ".metadata")
	return os.WriteFile(markerPath, []byte(model.Name), 0644)
}

// loadMetadata loads model metadata from the installation directory
func (i *ModelInstaller) loadMetadata(modelDir string) (*types.ModelMetadata, error) {
	// Implementation would load metadata from JSON
	// For now, just return basic info
	markerPath := filepath.Join(modelDir, ".metadata")
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return nil, err
	}

	return &types.ModelMetadata{
		Name: string(data),
	}, nil
}

// sanitizeModelName converts a model name to a safe directory name
func sanitizeModelName(name string) string {
	// Replace unsafe characters
	safe := name
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, unsafeChar := range unsafe {
		safe = strings.ReplaceAll(safe, unsafeChar, "-")
	}
	safe = filepath.ToSlash(safe)
	safe = filepath.Base(safe)
	return safe
}

// Made with Bob