package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// Infer performs inference on the remote cluster
func (rc *RemoteClient) Infer(ctx context.Context, req *engine.InferenceRequest) (*engine.InferenceResponse, error) {
	// Convert to OpenAI-compatible format
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Determine endpoint based on request type
	endpoint := rc.config.ClusterURL + "/v1/chat/completions"
	if req.Prompt != "" {
		endpoint = rc.config.ClusterURL + "/v1/completions"
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
