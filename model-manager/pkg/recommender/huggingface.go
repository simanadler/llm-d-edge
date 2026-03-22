package recommender

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/simanadler/llm-d-edge/model-manager/pkg/types"
)

// HuggingFaceRegistry queries the HuggingFace API for model information
type HuggingFaceRegistry struct {
	apiURL     string
	httpClient *http.Client
}

// NewHuggingFaceRegistry creates a new HuggingFace API client
func NewHuggingFaceRegistry() *HuggingFaceRegistry {
	return &HuggingFaceRegistry{
		apiURL: "https://huggingface.co/api",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the registry name
func (r *HuggingFaceRegistry) Name() string {
	return "huggingface"
}

// HFModel represents a model from HuggingFace API
type HFModel struct {
	ID          string   `json:"id"`
	ModelID     string   `json:"modelId"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	Downloads   int      `json:"downloads"`
	Likes       int      `json:"likes"`
	Private     bool     `json:"private"`
	Gated       bool     `json:"gated"`
	LibraryName string   `json:"library_name"`
}

// GetModels returns popular models from HuggingFace
func (r *HuggingFaceRegistry) GetModels(ctx context.Context) ([]types.ModelMetadata, error) {
	return r.GetPopularModels(ctx)
}

// GetModel retrieves metadata for a specific model
func (r *HuggingFaceRegistry) GetModel(ctx context.Context, modelID string) (*types.ModelMetadata, error) {
	return r.getModelMetadata(modelID), nil
}

// SearchModels searches for models on HuggingFace
func (r *HuggingFaceRegistry) SearchModels(ctx context.Context, query string, limit int) ([]types.ModelMetadata, error) {
	// Search for popular instruction-tuned models
	url := fmt.Sprintf("%s/models?search=%s&filter=text-generation&sort=downloads&direction=-1&limit=%d",
		r.apiURL, query, limit)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query HuggingFace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HuggingFace API error: %s - %s", resp.Status, string(body))
	}

	var hfModels []HFModel
	if err := json.NewDecoder(resp.Body).Decode(&hfModels); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our model metadata format
	models := make([]types.ModelMetadata, 0, len(hfModels))
	for _, hfModel := range hfModels {
		if metadata := r.convertToMetadata(hfModel); metadata != nil {
			models = append(models, *metadata)
		}
	}

	return models, nil
}

// GetPopularModels returns a curated list of popular models for different sizes
func (r *HuggingFaceRegistry) GetPopularModels(ctx context.Context) ([]types.ModelMetadata, error) {
	// Define popular models across different size categories
	popularRepos := []string{
		// Small models (0.5B-1B)
		"Qwen/Qwen2.5-0.5B-Instruct",
		"HuggingFaceTB/SmolLM-1.7B-Instruct",
		
		// Medium models (3B)
		"Qwen/Qwen2.5-3B-Instruct",
		"meta-llama/Llama-3.2-3B-Instruct",
		"microsoft/Phi-3-mini-4k-instruct",
		"stabilityai/stablelm-3b-4e1t",
		
		// Large models (7B)
		"mistralai/Mistral-7B-Instruct-v0.3",
		"meta-llama/Llama-3.1-8B-Instruct",
		"Qwen/Qwen2.5-7B-Instruct",
		
		// Very large models (13B+)
		"meta-llama/Llama-3.1-70B-Instruct",
		"Qwen/Qwen2.5-14B-Instruct",
		"mistralai/Mixtral-8x7B-Instruct-v0.1",
		"Qwen/Qwen2.5-32B-Instruct",
		"Qwen/Qwen2.5-72B-Instruct",
	}

	models := make([]types.ModelMetadata, 0, len(popularRepos))
	for _, repo := range popularRepos {
		metadata := r.getModelMetadata(repo)
		if metadata != nil {
			models = append(models, *metadata)
		}
	}

	return models, nil
}

// convertToMetadata converts HuggingFace model to our metadata format
func (r *HuggingFaceRegistry) convertToMetadata(hfModel HFModel) *types.ModelMetadata {
	// Extract model name and parameter count from ID
	parts := strings.Split(hfModel.ModelID, "/")
	if len(parts) != 2 {
		return nil
	}

	name := parts[1]
	paramCount := r.extractParamCount(name)
	if paramCount == "" {
		return nil // Skip models without clear parameter count
	}

	// Determine quality tier from tags
	qualityTier := "base"
	for _, tag := range hfModel.Tags {
		if strings.Contains(strings.ToLower(tag), "instruct") {
			qualityTier = "instruct"
			break
		} else if strings.Contains(strings.ToLower(tag), "chat") {
			qualityTier = "chat"
			break
		}
	}

	// Estimate download size (rough approximation)
	downloadSize := r.estimateDownloadSize(paramCount)

	return &types.ModelMetadata{
		Name:            name,
		ParameterCount:  paramCount,
		ContextLength:   r.estimateContextLength(name),
		ModelFamily:     r.extractFamily(name),
		QualityTier:     qualityTier,
		Tasks:           r.inferTasks(hfModel.Tags),
		Domains:         map[string]float64{"general": 0.8},
		License:         r.inferLicense(hfModel.Author),
		DownloadSizeGB:  downloadSize,
		Formats:         []string{"mlx", "gguf"},
		Quantizations:   []string{"q4", "q8", "fp16"},
		HuggingFaceRepo: hfModel.ModelID,
	}
}

// getModelMetadata returns metadata for a specific model repo
func (r *HuggingFaceRegistry) getModelMetadata(repo string) *types.ModelMetadata {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return nil
	}

	name := parts[1]
	paramCount := r.extractParamCount(name)
	
	// Determine quality tier
	qualityTier := "instruct"
	if strings.Contains(strings.ToLower(name), "chat") {
		qualityTier = "chat"
	}

	// Determine model family
	family := r.extractFamily(name)
	
	// Set tasks based on model type
	tasks := map[string]float64{
		"general": 0.85,
		"chat":    0.9,
	}
	
	if strings.Contains(strings.ToLower(name), "code") {
		tasks["code"] = 0.95
	}

	return &types.ModelMetadata{
		Name:            name,
		ParameterCount:  paramCount,
		ContextLength:   r.estimateContextLength(name),
		ModelFamily:     family,
		QualityTier:     qualityTier,
		Tasks:           tasks,
		Domains:         map[string]float64{"general": 0.9},
		License:         r.inferLicense(parts[0]),
		DownloadSizeGB:  r.estimateDownloadSize(paramCount),
		Formats:         []string{"mlx", "gguf"},
		Quantizations:   []string{"q4", "q8", "fp16"},
		HuggingFaceRepo: repo,
	}
}

// extractParamCount extracts parameter count from model name
func (r *HuggingFaceRegistry) extractParamCount(name string) string {
	name = strings.ToLower(name)
	
	// Common patterns
	patterns := []string{
		"0.5b", "1b", "1.7b", "3b", "7b", "8b", "13b", "14b", "30b", "32b", "70b", "72b",
	}
	
	for _, pattern := range patterns {
		if strings.Contains(name, pattern) {
			return strings.ToUpper(pattern)
		}
	}
	
	// Check for Mixtral pattern (8x7B)
	if strings.Contains(name, "8x7b") {
		return "7B" // Treat as 7B for compatibility
	}
	
	return ""
}

// extractFamily extracts model family from name
func (r *HuggingFaceRegistry) extractFamily(name string) string {
	name = strings.ToLower(name)
	
	families := map[string]string{
		"llama":   "llama",
		"mistral": "mistral",
		"mixtral": "mistral",
		"qwen":    "qwen",
		"phi":     "phi",
		"gemma":   "gemma",
		"smol":    "smol",
		"stable":  "stablelm",
	}
	
	for key, family := range families {
		if strings.Contains(name, key) {
			return family
		}
	}
	
	return "unknown"
}

// estimateContextLength estimates context length from model name
func (r *HuggingFaceRegistry) estimateContextLength(name string) int {
	name = strings.ToLower(name)
	
	// Check for explicit context length in name
	if strings.Contains(name, "32k") || strings.Contains(name, "32768") {
		return 32768
	}
	if strings.Contains(name, "128k") {
		return 131072
	}
	if strings.Contains(name, "4k") {
		return 4096
	}
	
	// Defaults by family
	if strings.Contains(name, "qwen") {
		return 32768
	}
	if strings.Contains(name, "llama-3") {
		return 8192
	}
	if strings.Contains(name, "mistral") {
		return 32768
	}
	
	return 8192 // Default
}

// estimateDownloadSize estimates download size based on parameter count
func (r *HuggingFaceRegistry) estimateDownloadSize(paramCount string) float64 {
	sizes := map[string]float64{
		"0.5B": 0.5,
		"1B":   1.0,
		"1.7B": 1.5,
		"3B":   2.0,
		"7B":   4.5,
		"8B":   5.0,
		"13B":  8.0,
		"14B":  9.0,
		"30B":  18.0,
		"32B":  20.0,
		"70B":  40.0,
		"72B":  45.0,
	}
	
	if size, ok := sizes[paramCount]; ok {
		return size
	}
	
	return 5.0 // Default
}

// inferTasks infers task capabilities from tags
func (r *HuggingFaceRegistry) inferTasks(tags []string) map[string]float64 {
	tasks := map[string]float64{
		"general": 0.8,
	}
	
	for _, tag := range tags {
		tag = strings.ToLower(tag)
		if strings.Contains(tag, "code") {
			tasks["code"] = 0.9
		}
		if strings.Contains(tag, "chat") {
			tasks["chat"] = 0.9
		}
		if strings.Contains(tag, "instruct") {
			tasks["general"] = 0.85
		}
	}
	
	return tasks
}

// inferLicense infers license from author
func (r *HuggingFaceRegistry) inferLicense(author string) string {
	author = strings.ToLower(author)
	
	licenses := map[string]string{
		"meta-llama":  "llama3",
		"mistralai":   "apache-2.0",
		"qwen":        "apache-2.0",
		"microsoft":   "mit",
		"google":      "apache-2.0",
		"stabilityai": "apache-2.0",
	}
	
	for key, license := range licenses {
		if strings.Contains(author, key) {
			return license
		}
	}
	
	return "unknown"
}

// Made with Bob
