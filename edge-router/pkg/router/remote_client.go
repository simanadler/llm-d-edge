package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/llm-d-incubation/llm-d-edge/pkg/config"
	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
	"go.uber.org/zap"
)

// RemoteClient handles communication with the remote llm-d cluster
type RemoteClient struct {
	config     config.RemoteClusterConfig
	httpClient *http.Client
	logger     *zap.Logger
}

// NewRemoteClient creates a new remote client
func NewRemoteClient(cfg config.RemoteClusterConfig, logger *zap.Logger) (*RemoteClient, error) {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &RemoteClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}, nil
}

// getModelNameWithoutProvider extracts the model name without the provider prefix
// Example: "Qwen/Qwen2.5-72B-Instruct" -> "qwen2-5-72b-instruct"
func getModelNameWithoutProvider(model string) string {
	// Split by "/" to remove provider prefix
	parts := strings.Split(model, "/")
	modelName := model
	if len(parts) > 1 {
		modelName = parts[len(parts)-1]
	}
	
	// Convert to lowercase and replace dots with dashes
	modelName = strings.ToLower(modelName)
	modelName = strings.ReplaceAll(modelName, ".", "-")
	
	return modelName
}

// buildClusterURL builds the cluster URL, adding model name for RITS endpoints
func (rc *RemoteClient) buildClusterURL(model string) string {
	baseURL := rc.config.ClusterURL
	
	// Check if "rits" is in the cluster URL
	if strings.Contains(strings.ToLower(baseURL), "rits") {
		// Extract model name without provider prefix
		modelName := getModelNameWithoutProvider(model)
		
		// Remove any trailing slashes from base URL
		baseURL = strings.TrimRight(baseURL, "/")
		
		// Check if model name is already in the URL
		if !strings.HasSuffix(strings.ToLower(baseURL), strings.ToLower(modelName)) {
			baseURL = baseURL + "/" + modelName
		}
	}
	
	return baseURL
}

// Infer performs inference on the remote cluster
func (rc *RemoteClient) Infer(ctx context.Context, req *engine.InferenceRequest) (*engine.InferenceResponse, error) {
	// Convert to OpenAI-compatible format
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build cluster URL with model name if needed (for RITS)
	clusterURL := rc.buildClusterURL(req.Model)
	
	// Determine endpoint based on request type
	endpoint := clusterURL + "/v1/chat/completions"
	if req.Prompt != "" {
		endpoint = clusterURL + "/v1/completions"
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if rc.config.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+rc.config.AuthToken)
	}
	//httpReq.Header.Set("RITS_API_KEY", "c6624bc75b20cf393d6cf7a9284e7db4")
	
	// Add custom headers from config
	for k, v := range rc.config.Headers {
		httpReq.Header.Set(k, v)
	}

	// Send request
	rc.logger.Debug("sending request to remote cluster",
		zap.String("endpoint", endpoint),
		zap.String("model", req.Model),
	)
	
	resp, err := rc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("remote cluster returned error: %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response engine.InferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// Close closes the remote client
func (rc *RemoteClient) Close() error {
	rc.httpClient.CloseIdleConnections()
	return nil
}

// Made with Bob
