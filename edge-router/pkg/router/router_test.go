package router

import (
	"context"
	"testing"
	"time"

	"github.com/llm-d-incubation/llm-d-edge/pkg/config"
	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
	"go.uber.org/zap"
)

// MockEngine implements the InferenceEngine interface for testing
type MockEngine struct {
	healthy       bool
	state         engine.EngineState
	loadedModel   string
	inferenceFunc func(ctx context.Context, req engine.InferenceRequest) (*engine.InferenceResponse, error)
}

func NewMockEngine() *MockEngine {
	return &MockEngine{
		healthy: true,
		state:   engine.StateReady,
	}
}

func (m *MockEngine) Initialize(ctx context.Context, config engine.EngineConfig) error {
	return nil
}

func (m *MockEngine) LoadModel(ctx context.Context, modelPath string) error {
	m.loadedModel = modelPath
	m.state = engine.StateReady
	return nil
}

func (m *MockEngine) Infer(ctx context.Context, request engine.InferenceRequest) (*engine.InferenceResponse, error) {
	if m.inferenceFunc != nil {
		return m.inferenceFunc(ctx, request)
	}

	return &engine.InferenceResponse{
		ID:      "test-123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   request.Model,
		Choices: []engine.Choice{
			{
				Index: 0,
				Message: engine.Message{
					Role:    "assistant",
					Content: "Test response",
				},
				FinishReason: "stop",
			},
		},
		Usage: engine.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

func (m *MockEngine) Unload(ctx context.Context) error {
	m.loadedModel = ""
	m.state = engine.StateIdle
	return nil
}

func (m *MockEngine) GetCapabilities() engine.EngineCapabilities {
	return engine.EngineCapabilities{
		Name:              "mock",
		Version:           "1.0.0",
		SupportedFormats:  []string{"mock"},
		MaxContextLength:  4096,
		SupportsStreaming: true,
		SupportsChat:      true,
		GPUAcceleration:   false,
		Platform:          "test",
	}
}

func (m *MockEngine) GetStatus() engine.EngineStatus {
	return engine.EngineStatus{
		State:            m.state,
		LoadedModel:      m.loadedModel,
		MemoryUsageMB:    1024,
		TotalInferences:  0,
		AverageLatencyMs: 100,
	}
}

func (m *MockEngine) IsHealthy() bool {
	return m.healthy
}

// Test helper to create a test configuration
func createTestConfig(policy config.RoutingPolicy) *config.Config {
	return &config.Config{
		Edge: config.EdgeConfig{
			Platform: "test",
			Routing: config.RoutingConfig{
				Policy:   policy,
				Fallback: "remote",
			},
			Models: config.ModelsConfig{
				Local: []config.ExtendedLocalModelConfig{
					{
						LocalModelConfig: config.LocalModelConfig{
							Name:         "test-model",
							Format:       "mock",
							Quantization: "4bit",
							Priority:     1,
						},
					},
				},
				Remote: config.RemoteClusterConfig{
					ClusterURL: "http://test-cluster.com",
					AuthToken:  "test-token",
					Timeout:    60,
				},
			},
			RoutingRules: []config.RoutingRule{},
		},
	}
}

func TestRouterCreation(t *testing.T) {
	cfg := createTestConfig(config.PolicyLocalFirst)
	mockEngine := NewMockEngine()
	logger := zap.NewNop()

	router, err := NewRouter(cfg, mockEngine, logger)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	if router == nil {
		t.Fatal("Router is nil")
	}

	if router.config != cfg {
		t.Error("Router config not set correctly")
	}

	if router.localEngine != mockEngine {
		t.Error("Router local engine not set correctly")
	}
}

func TestLocalFirstPolicy(t *testing.T) {
	cfg := createTestConfig(config.PolicyLocalFirst)
	mockEngine := NewMockEngine()
	logger := zap.NewNop()

	router, err := NewRouter(cfg, mockEngine, logger)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	req := &engine.InferenceRequest{
		Model: "test-model",
		Messages: []engine.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
	}

	decision, err := router.Route(context.Background(), req, true)
	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if decision.Target != TargetLocal {
		t.Errorf("Expected local target, got %s", decision.Target)
	}

	if decision.Reason != "local_first_policy" {
		t.Errorf("Expected 'local_first_policy' reason, got %s", decision.Reason)
	}
}

func TestRemoteFirstPolicy(t *testing.T) {
	cfg := createTestConfig(config.PolicyRemoteFirst)
	mockEngine := NewMockEngine()
	logger := zap.NewNop()

	router, err := NewRouter(cfg, mockEngine, logger)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	req := &engine.InferenceRequest{
		Model: "test-model",
		Messages: []engine.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	decision, err := router.Route(context.Background(), req, true)
	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if decision.Target != TargetRemote {
		t.Errorf("Expected remote target, got %s", decision.Target)
	}

	if decision.Reason != "remote_first_policy" {
		t.Errorf("Expected 'remote_first_policy' reason, got %s", decision.Reason)
	}
}

func TestHybridPolicySmallRequest(t *testing.T) {
	cfg := createTestConfig(config.PolicyHybrid)
	mockEngine := NewMockEngine()
	logger := zap.NewNop()

	router, err := NewRouter(cfg, mockEngine, logger)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Small request (< 1000 tokens)
	req := &engine.InferenceRequest{
		Model: "test-model",
		Messages: []engine.Message{
			{Role: "user", Content: "Short message"},
		},
	}

	decision, err := router.Route(context.Background(), req, true)
	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if decision.Target != TargetLocal {
		t.Errorf("Expected local target for small request, got %s", decision.Target)
	}
}

func TestHybridPolicyLargeRequest(t *testing.T) {
	cfg := createTestConfig(config.PolicyHybrid)
	mockEngine := NewMockEngine()
	logger := zap.NewNop()

	router, err := NewRouter(cfg, mockEngine, logger)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Large request (> 1000 tokens)
	largeContent := ""
	for i := 0; i < 5000; i++ {
		largeContent += "word "
	}

	req := &engine.InferenceRequest{
		Model: "test-model",
		Messages: []engine.Message{
			{Role: "user", Content: largeContent},
		},
	}

	decision, err := router.Route(context.Background(), req, true)
	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if decision.Target != TargetRemote {
		t.Errorf("Expected remote target for large request, got %s", decision.Target)
	}
}

func TestModelNotAvailableLocally(t *testing.T) {
	cfg := createTestConfig(config.PolicyLocalFirst)
	mockEngine := NewMockEngine()
	logger := zap.NewNop()

	router, err := NewRouter(cfg, mockEngine, logger)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Request for model not in local config
	req := &engine.InferenceRequest{
		Model: "non-existent-model",
		Messages: []engine.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	decision, err := router.Route(context.Background(), req, false)
	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if decision.Target != TargetRemote {
		t.Errorf("Expected remote target for unavailable model, got %s", decision.Target)
	}

	if decision.Reason != "model_not_available_locally" {
		t.Errorf("Expected 'model_not_available_locally' reason, got %s", decision.Reason)
	}
}

func TestRoutingRules(t *testing.T) {
	cfg := createTestConfig(config.PolicyLocalFirst)
	cfg.Edge.RoutingRules = []config.RoutingRule{
		{
			Condition: "prompt_tokens < 1000 AND model IN local_models",
			Action:    "route_local",
		},
		{
			Condition: "prompt_tokens >= 1000",
			Action:    "route_remote",
		},
	}

	mockEngine := NewMockEngine()
	logger := zap.NewNop()

	router, err := NewRouter(cfg, mockEngine, logger)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Test small request - should match first rule
	smallReq := &engine.InferenceRequest{
		Model: "test-model",
		Messages: []engine.Message{
			{Role: "user", Content: "Short"},
		},
	}

	decision, err := router.Route(context.Background(), smallReq, true)
	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if decision.Target != TargetLocal {
		t.Errorf("Expected local target for small request, got %s", decision.Target)
	}
}

func TestEstimatePromptTokens(t *testing.T) {
	cfg := createTestConfig(config.PolicyLocalFirst)
	mockEngine := NewMockEngine()
	logger := zap.NewNop()

	router, err := NewRouter(cfg, mockEngine, logger)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	tests := []struct {
		name     string
		request  *engine.InferenceRequest
		expected int
	}{
		{
			name: "Direct prompt",
			request: &engine.InferenceRequest{
				Prompt: "This is a test prompt",
			},
			expected: 5, // ~20 chars / 4
		},
		{
			name: "Messages",
			request: &engine.InferenceRequest{
				Messages: []engine.Message{
					{Role: "user", Content: "Hello world"},
				},
			},
			expected: 2, // ~11 chars / 4
		},
		{
			name: "Multiple messages",
			request: &engine.InferenceRequest{
				Messages: []engine.Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
				},
			},
			expected: 5, // ~20 chars / 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := router.estimatePromptTokens(tt.request)
			if tokens != tt.expected {
				t.Errorf("Expected %d tokens, got %d", tt.expected, tokens)
			}
		})
	}
}

func TestCanServeLocally(t *testing.T) {
	cfg := createTestConfig(config.PolicyLocalFirst)
	logger := zap.NewNop()

	tests := []struct {
		name     string
		engine   *MockEngine
		expected bool
	}{
		{
			name: "Healthy engine",
			engine: &MockEngine{
				healthy: true,
				state:   engine.StateReady,
			},
			expected: true,
		},
		{
			name: "Unhealthy engine",
			engine: &MockEngine{
				healthy: false,
				state:   engine.StateError,
			},
			expected: false,
		},
		{
			name: "Loading engine",
			engine: &MockEngine{
				healthy: true,
				state:   engine.StateLoading,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, err := NewRouter(cfg, tt.engine, logger)
			if err != nil {
				t.Fatalf("Failed to create router: %v", err)
			}

			req := &engine.InferenceRequest{
				Model: "test-model",
			}

			canServe := router.canServeLocally(req)
			if canServe != tt.expected {
				t.Errorf("Expected canServeLocally=%v, got %v", tt.expected, canServe)
			}
		})
	}
}

func TestMobileOptimizedPolicy(t *testing.T) {
	cfg := createTestConfig(config.PolicyMobileOptimized)
	mockEngine := NewMockEngine()
	logger := zap.NewNop()

	router, err := NewRouter(cfg, mockEngine, logger)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Small request should use local
	smallReq := &engine.InferenceRequest{
		Model: "test-model",
		Messages: []engine.Message{
			{Role: "user", Content: "Hi"},
		},
	}

	decision, err := router.Route(context.Background(), smallReq, true)
	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if decision.Target != TargetLocal {
		t.Errorf("Expected local target for small mobile request, got %s", decision.Target)
	}

	// Large request should use remote
	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += "word "
	}

	largeReq := &engine.InferenceRequest{
		Model: "test-model",
		Messages: []engine.Message{
			{Role: "user", Content: largeContent},
		},
	}

	decision, err = router.Route(context.Background(), largeReq, true)
	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if decision.Target != TargetRemote {
		t.Errorf("Expected remote target for large mobile request, got %s", decision.Target)
	}
}

func TestLatencyOptimizedPolicy(t *testing.T) {
	cfg := createTestConfig(config.PolicyLatencyOptimized)
	mockEngine := NewMockEngine()
	logger := zap.NewNop()

	router, err := NewRouter(cfg, mockEngine, logger)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Streaming request with small prompt should use local
	streamReq := &engine.InferenceRequest{
		Model:  "test-model",
		Stream: true,
		Messages: []engine.Message{
			{Role: "user", Content: "Quick question"},
		},
	}

	decision, err := router.Route(context.Background(), streamReq, true)
	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if decision.Target != TargetLocal {
		t.Errorf("Expected local target for streaming request, got %s", decision.Target)
	}

	// Non-streaming request should use remote
	batchReq := &engine.InferenceRequest{
		Model:  "test-model",
		Stream: false,
		Messages: []engine.Message{
			{Role: "user", Content: "Batch processing"},
		},
	}

	decision, err = router.Route(context.Background(), batchReq, true)
	if err != nil {
		t.Fatalf("Routing failed: %v", err)
	}

	if decision.Target != TargetRemote {
		t.Errorf("Expected remote target for batch request, got %s", decision.Target)
	}
}

// Made with Bob
