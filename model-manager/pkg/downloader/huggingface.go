package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// ModelConverter is an interface for converting models
type ModelConverter interface {
	Convert(ctx context.Context, model types.ModelMetadata, destPath string, opts ConvertOptions) error
	SupportsFormat(format string) bool
	IsAvailable() error
	Name() string
}

// ConvertOptions contains options for model conversion
type ConvertOptions struct {
	TargetFormat      string
	Quantize          bool
	QuantizationBits  int
	QuantizationGroup int
	ProgressCallback  func(message string)
}

// HuggingFaceDownloader downloads models from HuggingFace
type HuggingFaceDownloader struct {
	httpClient *http.Client
	baseURL    string
	apiURL     string
	converter  ModelConverter
}

// HFRepoFile represents a file in a HuggingFace repository
type HFRepoFile struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// NewHuggingFaceDownloader creates a new HuggingFace downloader
func NewHuggingFaceDownloader() *HuggingFaceDownloader {
	return &HuggingFaceDownloader{
		httpClient: &http.Client{
			Timeout: 0, // No timeout for large downloads
		},
		baseURL: "https://huggingface.co",
		apiURL:  "https://huggingface.co/api",
	}
}

// Name returns the downloader name
func (d *HuggingFaceDownloader) Name() string {
	return "huggingface"
}

// SupportsModel checks if this downloader supports the given model
func (d *HuggingFaceDownloader) SupportsModel(model types.ModelMetadata) bool {
	return model.HuggingFaceRepo != ""
}

// Download downloads a model from HuggingFace
func (d *HuggingFaceDownloader) Download(ctx context.Context, model types.ModelMetadata, dest string, opts DownloadOptions) error {
	if !d.SupportsModel(model) {
		return fmt.Errorf("model %s is not available on HuggingFace", model.Name)
	}

	// Find the best matching file in the repository
	file, err := d.findBestFile(ctx, model, opts)
	if err != nil {
		return fmt.Errorf("failed to find suitable file: %w", err)
	}

	// Construct download URL
	url := fmt.Sprintf("%s/%s/resolve/main/%s", d.baseURL, model.HuggingFaceRepo, file.Path)

	// Create destination directory
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Determine filename from the file path
	filename := filepath.Base(file.Path)
	destPath := filepath.Join(dest, filename)

	// Check if file already exists and resume is enabled
	var startByte int64
	if opts.Resume {
		if info, err := os.Stat(destPath); err == nil {
			startByte = info.Size()
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add range header for resume
	if startByte > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
	}

	// Execute request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Get total size
	totalSize := resp.ContentLength
	if startByte > 0 {
		totalSize += startByte
	}

	// Open destination file
	fileFlag := os.O_CREATE | os.O_WRONLY
	if startByte > 0 {
		fileFlag |= os.O_APPEND
	} else {
		fileFlag |= os.O_TRUNC
	}

	outFile, err := os.OpenFile(destPath, fileFlag, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Download with progress tracking
	var writer io.Writer = outFile
	if opts.ProgressCallback != nil {
		writer = NewProgressWriter(outFile, totalSize, opts.ProgressCallback)
		
		// Send initial progress
		opts.ProgressCallback(DownloadProgress{
			BytesDownloaded: startByte,
			TotalBytes:      totalSize,
			Percentage:      float64(startByte) / float64(totalSize) * 100,
			Status:          "starting",
		})
	}

	// Copy data
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Send completion progress
	if opts.ProgressCallback != nil {
		opts.ProgressCallback(DownloadProgress{
			BytesDownloaded: totalSize,
			TotalBytes:      totalSize,
			Percentage:      100,
			Status:          "complete",
		})
	}

	return nil
}

// GetDownloadURL returns the download URL for a model
func (d *HuggingFaceDownloader) GetDownloadURL(model types.ModelMetadata, opts DownloadOptions) (string, error) {
	if model.HuggingFaceRepo == "" {
		return "", fmt.Errorf("model has no HuggingFace repository")
	}

	// Find the best matching file
	file, err := d.findBestFile(context.Background(), model, opts)
	if err != nil {
		return "", fmt.Errorf("failed to find suitable file: %w", err)
	}

	// Construct URL
	url := fmt.Sprintf("%s/%s/resolve/main/%s", d.baseURL, model.HuggingFaceRepo, file.Path)
	return url, nil
}

// findBestFile queries the HuggingFace API to find the best matching file
func (d *HuggingFaceDownloader) findBestFile(ctx context.Context, model types.ModelMetadata, opts DownloadOptions) (*HFRepoFile, error) {
	// Query the repository tree
	url := fmt.Sprintf("%s/models/%s/tree/main", d.apiURL, model.HuggingFaceRepo)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	var files []HFRepoFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Filter files based on format and quantization preferences
	format := strings.ToLower(opts.Format)
	if format == "" {
		format = "gguf" // Default to GGUF as it's most compatible
	}

	quant := strings.ToLower(opts.Quantization)
	if quant == "" {
		quant = "q4" // Default to 4-bit
	}

	// Score each file and find the best match
	var bestFile *HFRepoFile
	bestScore := 0

	for i := range files {
		file := &files[i]
		if file.Type != "file" {
			continue
		}

		score := d.scoreFile(file.Path, format, quant)
		if score > bestScore {
			bestScore = score
			bestFile = file
		}
	}

	if bestFile == nil {
		return nil, fmt.Errorf("no suitable file found for format=%s, quantization=%s", format, quant)
	}

	return bestFile, nil
}

// scoreFile scores how well a file matches the desired format and quantization
func (d *HuggingFaceDownloader) scoreFile(path, desiredFormat, desiredQuant string) int {
	path = strings.ToLower(path)
	score := 0

	// Exclude non-model files
	excludePatterns := []string{
		".gitattributes", ".gitignore", "readme", "license", ".txt", ".md",
		".json", ".yaml", ".yml", "config", ".py", ".sh",
	}
	for _, pattern := range excludePatterns {
		if strings.Contains(path, pattern) {
			return -1000 // Heavily penalize non-model files
		}
	}

	// Check format match
	if desiredFormat == "gguf" && strings.HasSuffix(path, ".gguf") {
		score += 100
	} else if desiredFormat == "mlx" && strings.Contains(path, "mlx") && strings.HasSuffix(path, ".safetensors") {
		score += 100
	} else if desiredFormat == "safetensors" && strings.HasSuffix(path, ".safetensors") {
		score += 100
	}

	// If no format match, return low score
	if score == 0 {
		return -100
	}

	// Check quantization match
	quantPatterns := map[string][]string{
		"q4":   {"q4_0", "q4_k", "q4_k_m", "q4"},
		"q8":   {"q8_0", "q8"},
		"fp16": {"f16", "fp16"},
	}

	if patterns, ok := quantPatterns[desiredQuant]; ok {
		for _, pattern := range patterns {
			if strings.Contains(path, pattern) {
				score += 50
				break
			}
		}
	}

	// Prefer files in root or common directories
	if !strings.Contains(path, "/") || strings.HasPrefix(path, "gguf/") {
		score += 10
	}

	// Avoid certain patterns
	if strings.Contains(path, "imatrix") || strings.Contains(path, "test") {
		score -= 20
	}

	return score
}

// Made with Bob