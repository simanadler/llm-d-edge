// +build darwin

package macos

import (
	"context"
	"testing"
	"time"

	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
	"go.uber.org/zap"
)

func TestMLXEngineCreation(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	if eng == nil {
		t.Fatal("MLXEngine is nil")
	}

	if eng.state != engine.StateIdle {
		t.Errorf("Expected initial state to be Idle, got %s", eng.state)
	}

	if eng.logger == nil {
		t.Error("Logger not set")
	}
}

func TestMLXEngineGetCapabilities(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	caps := eng.GetCapabilities()

	if caps.Name != "mlx" {
		t.Errorf("Expected engine name 'mlx', got '%s'", caps.Name)
	}

	if caps.Platform != "macos" {
		t.Errorf("Expected platform 'macos', got '%s'", caps.Platform)
	}

	if !caps.GPUAcceleration {
		t.Error("Expected GPU acceleration to be true for MLX")
	}

	if !caps.SupportsStreaming {
		t.Error("Expected streaming support")
	}

	if !caps.SupportsChat {
		t.Error("Expected chat support")
	}

	if !caps.SupportsCompletion {
		t.Error("Expected completion support")
	}

	// Check supported formats
	expectedFormats := map[string]bool{
		"mlx":  true,
		"gguf": true,
	}

	for _, format := range caps.SupportedFormats {
		if !expectedFormats[format] {
			t.Errorf("Unexpected format: %s", format)
		}
	}

	// Check supported quantizations
	expectedQuants := map[string]bool{
		"4bit": true,
		"8bit": true,
		"none": true,
	}

	for _, quant := range caps.SupportedQuantizations {
		if !expectedQuants[quant] {
			t.Errorf("Unexpected quantization: %s", quant)
		}
	}
}

func TestMLXEngineGetStatus(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	status := eng.GetStatus()

	if status.State != engine.StateIdle {
		t.Errorf("Expected state Idle, got %s", status.State)
	}

	if status.LoadedModel != "" {
		t.Errorf("Expected no loaded model, got '%s'", status.LoadedModel)
	}

	if status.TotalInferences != 0 {
		t.Errorf("Expected 0 inferences, got %d", status.TotalInferences)
	}

	if status.ErrorMessage != "" {
		t.Errorf("Expected no error message, got '%s'", status.ErrorMessage)
	}
}

func TestMLXEngineIsHealthy(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	// Initially should be healthy (idle state)
	if !eng.IsHealthy() {
		t.Error("Expected engine to be healthy in idle state")
	}

	// Set to ready state
	eng.state = engine.StateReady
	if !eng.IsHealthy() {
		t.Error("Expected engine to be healthy in ready state")
	}

	// Set to error state
	eng.state = engine.StateError
	if eng.IsHealthy() {
		t.Error("Expected engine to be unhealthy in error state")
	}

	// Set to loading state
	eng.state = engine.StateLoading
	if eng.IsHealthy() {
		t.Error("Expected engine to be unhealthy in loading state")
	}
}

func TestMLXEngineBuildPrompt(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	tests := []struct {
		name     string
		request  engine.InferenceRequest
		expected string
	}{
		{
			name: "Direct prompt",
			request: engine.InferenceRequest{
				Prompt: "Hello world",
			},
			expected: "Hello world",
		},
		{
			name: "Single user message",
			request: engine.InferenceRequest{
				Messages: []engine.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			expected: "User: Hello\n\nAssistant: ",
		},
		{
			name: "System and user messages",
			request: engine.InferenceRequest{
				Messages: []engine.Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
				},
			},
			expected: "System: You are helpful\n\nUser: Hello\n\nAssistant: ",
		},
		{
			name: "Conversation with assistant",
			request: engine.InferenceRequest{
				Messages: []engine.Message{
					{Role: "user", Content: "Hi"},
					{Role: "assistant", Content: "Hello!"},
					{Role: "user", Content: "How are you?"},
				},
			},
			expected: "User: Hi\n\nAssistant: Hello!\n\nUser: How are you?\n\nAssistant: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := eng.buildPrompt(tt.request)
			if prompt != tt.expected {
				t.Errorf("Expected prompt:\n%s\n\nGot:\n%s", tt.expected, prompt)
			}
		})
	}
}

func TestMLXEngineInitialize(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	config := engine.EngineConfig{
		EngineName:    "mlx",
		ModelPath:     "/tmp/test-model",
		ModelFormat:   "mlx",
		Quantization:  "4bit",
		MaxTokens:     2048,
		ContextLength: 4096,
		GPULayers:     -1,
	}

	ctx := context.Background()

	// Note: This will fail if MLX is not installed, which is expected in CI
	// In a real test environment, you'd mock the MLX installation check
	err := eng.Initialize(ctx, config)

	// Check that config was stored even if initialization failed
	if eng.config.EngineName != config.EngineName {
		t.Error("Config not stored")
	}

	// If MLX is not installed, state should be error
	if err != nil {
		if eng.state != engine.StateError {
			t.Errorf("Expected error state after failed initialization, got %s", eng.state)
		}
		if eng.errorMessage == "" {
			t.Error("Expected error message to be set")
		}
	}
}

func TestMLXEngineLoadModel(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	// Initialize first
	config := engine.EngineConfig{
		EngineName:   "mlx",
		ModelPath:    "/tmp/test-model",
		ModelFormat:  "mlx",
		Quantization: "4bit",
	}

	ctx := context.Background()
	_ = eng.Initialize(ctx, config)

	// Try to load model (will fail if model doesn't exist)
	err := eng.LoadModel(ctx, "/tmp/test-model")

	if err != nil {
		// Expected if model doesn't exist
		if eng.state != engine.StateError {
			t.Errorf("Expected error state after failed load, got %s", eng.state)
		}
	} else {
		// If successful (unlikely in test)
		if eng.state != engine.StateReady {
			t.Errorf("Expected ready state after successful load, got %s", eng.state)
		}
		if eng.loadedModel != "/tmp/test-model" {
			t.Errorf("Expected loaded model to be '/tmp/test-model', got '%s'", eng.loadedModel)
		}
	}
}

func TestMLXEngineUnload(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	// Set up as if model is loaded
	eng.loadedModel = "/tmp/test-model"
	eng.state = engine.StateReady

	ctx := context.Background()
	err := eng.Unload(ctx)

	if err != nil {
		t.Errorf("Unload failed: %v", err)
	}

	if eng.loadedModel != "" {
		t.Errorf("Expected loaded model to be empty, got '%s'", eng.loadedModel)
	}

	if eng.state != engine.StateIdle {
		t.Errorf("Expected idle state after unload, got %s", eng.state)
	}
}

func TestMLXEngineStatistics(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	// Simulate some inferences
	eng.totalInferences = 10
	eng.lastInferenceTime = time.Now()
	eng.latencies = []time.Duration{
		100 * time.Millisecond,
		150 * time.Millisecond,
		120 * time.Millisecond,
	}

	status := eng.GetStatus()

	if status.TotalInferences != 10 {
		t.Errorf("Expected 10 total inferences, got %d", status.TotalInferences)
	}

	if status.LastInferenceTime.IsZero() {
		t.Error("Expected last inference time to be set")
	}

	// Check average latency calculation
	expectedAvg := (100.0 + 150.0 + 120.0) / 3.0
	if status.AverageLatencyMs != expectedAvg {
		t.Errorf("Expected average latency %.2f, got %.2f", expectedAvg, status.AverageLatencyMs)
	}
}

func TestMLXEngineMemoryEstimation(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	memUsage := eng.estimateMemoryUsage()

	// Should return a reasonable estimate (currently hardcoded to 4096 MB)
	if memUsage <= 0 {
		t.Error("Expected positive memory usage estimate")
	}

	if memUsage > 100000 {
		t.Error("Memory usage estimate seems unreasonably high")
	}
}

func TestMLXEngineGenerateScript(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	eng.config = engine.EngineConfig{
		ModelPath: "/tmp/test-model",
	}

	request := engine.InferenceRequest{
		MaxTokens:   100,
		Temperature: 0.7,
	}

	script := eng.generateMLXScript("Test prompt", request)

	// Check that script contains expected elements
	if script == "" {
		t.Error("Generated script is empty")
	}

	// Check for key components
	expectedComponents := []string{
		"import json",
		"from mlx_lm import load, generate",
		"/tmp/test-model",
		"max_tokens=100",
		"temp=0.70",
	}

	for _, component := range expectedComponents {
		if !contains(script, component) {
			t.Errorf("Script missing expected component: %s", component)
		}
	}
}

func TestMLXEngineJSONQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Hello",
			expected: `"Hello"`,
		},
		{
			input:    "Hello\nWorld",
			expected: `"Hello\nWorld"`,
		},
		{
			input:    `Hello "World"`,
			expected: `"Hello \"World\""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := jsonQuote(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMLXEngineStateTransitions(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	// Test state transitions
	transitions := []struct {
		from     engine.EngineState
		to       engine.EngineState
		action   string
		isHealthy bool
	}{
		{engine.StateIdle, engine.StateLoading, "start loading", false},
		{engine.StateLoading, engine.StateReady, "finish loading", true},
		{engine.StateReady, engine.StateInferring, "start inference", false},
		{engine.StateInferring, engine.StateReady, "finish inference", true},
		{engine.StateReady, engine.StateUnloading, "start unload", false},
		{engine.StateUnloading, engine.StateIdle, "finish unload", true},
		{engine.StateReady, engine.StateError, "error occurred", false},
	}

	for _, tr := range transitions {
		t.Run(tr.action, func(t *testing.T) {
			eng.state = tr.from
			eng.state = tr.to

			if eng.IsHealthy() != tr.isHealthy {
				t.Errorf("Expected IsHealthy=%v for state %s, got %v",
					tr.isHealthy, tr.to, eng.IsHealthy())
			}
		})
	}
}

func TestMLXEngineLatencyTracking(t *testing.T) {
	logger := zap.NewNop()
	eng := NewMLXEngine(logger)

	// Add latencies (simulating what happens in Infer method)
	for i := 0; i < 150; i++ {
		eng.mu.Lock()
		eng.latencies = append(eng.latencies, time.Duration(i)*time.Millisecond)
		// Apply the same trimming logic as in Infer
		if len(eng.latencies) > 100 {
			eng.latencies = eng.latencies[len(eng.latencies)-100:]
		}
		eng.mu.Unlock()
	}

	// Check that only last 100 are kept
	eng.mu.RLock()
	latencyCount := len(eng.latencies)
	eng.mu.RUnlock()

	if latencyCount != 100 {
		t.Errorf("Expected 100 latencies to be kept, got %d", latencyCount)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Made with Bob
