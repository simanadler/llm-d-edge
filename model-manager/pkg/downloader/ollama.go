package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// OllamaDownloader downloads models using Ollama
type OllamaDownloader struct {
	httpClient *http.Client
	baseURL    string
}

// NewOllamaDownloader creates a new Ollama downloader
func NewOllamaDownloader() *OllamaDownloader {
	return &OllamaDownloader{
		httpClient: &http.Client{},
		baseURL:    "http://localhost:11434",
	}
}

// Name returns the downloader name
func (d *OllamaDownloader) Name() string {
	return "ollama"
}

// SupportsModel checks if this downloader supports the given model
func (d *OllamaDownloader) SupportsModel(model types.ModelMetadata) bool {
	// Ollama supports most popular models
	family := strings.ToLower(model.ModelFamily)
	supportedFamilies := []string{"llama", "mistral", "qwen", "phi", "gemma"}
	
	for _, supported := range supportedFamilies {
		if family == supported {
			return true
		}
	}
	
	return false
}

// Download downloads a model using Ollama
func (d *OllamaDownloader) Download(ctx context.Context, model types.ModelMetadata, dest string, opts DownloadOptions) error {
	if !d.SupportsModel(model) {
		return fmt.Errorf("model %s is not supported by Ollama", model.Name)
	}

	// Get Ollama model name
	ollamaModel := d.getOllamaModelName(model, opts)

	// Create pull request
	pullReq := map[string]interface{}{
		"name":   ollamaModel,
		"stream": true,
	}

	reqBody, err := json.Marshal(pullReq)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Send pull request
	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/api/pull", strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pull failed with status: %s", resp.Status)
	}

	// Parse streaming response
	decoder := json.NewDecoder(resp.Body)
	for {
		var progress map[string]interface{}
		if err := decoder.Decode(&progress); err != nil {
			break
		}

		// Send progress updates
		if opts.ProgressCallback != nil {
			status, _ := progress["status"].(string)
			completed, _ := progress["completed"].(float64)
			total, _ := progress["total"].(float64)

			if total > 0 {
				opts.ProgressCallback(DownloadProgress{
					BytesDownloaded: int64(completed),
					TotalBytes:      int64(total),
					Percentage:      (completed / total) * 100,
					Status:          status,
				})
			}
		}
	}

	return nil
}

// GetDownloadURL returns the download URL (not applicable for Ollama)
func (d *OllamaDownloader) GetDownloadURL(model types.ModelMetadata, opts DownloadOptions) (string, error) {
	return "", fmt.Errorf("Ollama does not use direct download URLs")
}

// getOllamaModelName converts our model metadata to Ollama model name
func (d *OllamaDownloader) getOllamaModelName(model types.ModelMetadata, opts DownloadOptions) string {
	family := strings.ToLower(model.ModelFamily)
	paramCount := strings.ToLower(model.ParameterCount)
	
	// Remove 'B' suffix
	paramCount = strings.TrimSuffix(paramCount, "b")
	
	// Get quantization
	quant := opts.Quantization
	if quant == "" {
		quant = "q4_0" // Default Ollama quantization
	} else {
		// Convert our quantization format to Ollama format
		quant = strings.ToLower(quant)
		if quant == "q4" {
			quant = "q4_0"
		} else if quant == "q8" {
			quant = "q8_0"
		}
	}
	
	// Build Ollama model name
	// Format: {family}:{size}{quant}
	return fmt.Sprintf("%s:%s%s", family, paramCount, quant)
}

// Made with Bob