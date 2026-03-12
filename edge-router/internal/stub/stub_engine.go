package stub

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
	"go.uber.org/zap"
)

// StubEngine implements the InferenceEngine interface for testing
// It simulates local inference without requiring actual models or hardware
type StubEngine struct {
	mu     sync.RWMutex
	logger *zap.Logger

	config       engine.EngineConfig
	state        engine.EngineState
	loadedModel  string
	errorMessage string

	// Configuration
	simulateLatencyMs int
	simulateErrors    bool
	errorRate         float64

	// Statistics
	totalInferences   int64
	lastInferenceTime time.Time
	latencies         []time.Duration
}

// NewStubEngine creates a new stub inference engine for testing
func NewStubEngine(logger *zap.Logger) *StubEngine {
	return &StubEngine{
		logger:            logger,
		state:             engine.StateIdle,
		latencies:         make([]time.Duration, 0, 100),
		simulateLatencyMs: 50,  // Default 50ms latency
		simulateErrors:    false,
		errorRate:         0.0,
	}
}

// NewStubEngineWithConfig creates a stub engine with custom configuration
func NewStubEngineWithConfig(logger *zap.Logger, latencyMs int, simulateErrors bool, errorRate float64) *StubEngine {
	eng := NewStubEngine(logger)
	eng.simulateLatencyMs = latencyMs
	eng.simulateErrors = simulateErrors
	eng.errorRate = errorRate
	return eng
}

// Initialize sets up the stub engine
func (e *StubEngine) Initialize(ctx context.Context, config engine.EngineConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Info("initializing stub engine",
		zap.String("engine", config.EngineName),
		zap.String("model_path", config.ModelPath),
		zap.Int("latency_ms", e.simulateLatencyMs),
	)

	// Store config first
	e.config = config

	// Simulate initialization delay
	time.Sleep(time.Duration(e.simulateLatencyMs) * time.Millisecond)

	// Simulate random initialization failure if configured
	if e.simulateErrors && rand.Float64() < e.errorRate {
		e.state = engine.StateError
		e.errorMessage = "simulated initialization error"
		return fmt.Errorf("stub engine initialization failed (simulated)")
	}

	e.state = engine.StateIdle
	e.errorMessage = ""

	e.logger.Info("stub engine initialized successfully")
	return nil
}

// LoadModel simulates loading a model into memory
func (e *StubEngine) LoadModel(ctx context.Context, modelPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Info("loading model (stub)", zap.String("model_path", modelPath))
	e.state = engine.StateLoading

	// Simulate loading delay
	time.Sleep(time.Duration(e.simulateLatencyMs*2) * time.Millisecond)

	// Simulate random loading failure if configured
	if e.simulateErrors && rand.Float64() < e.errorRate {
		e.state = engine.StateError
		e.errorMessage = "simulated model loading error"
		return fmt.Errorf("failed to load model (simulated): %s", modelPath)
	}

	// Update config with new model path
	e.config.ModelPath = modelPath
	e.loadedModel = modelPath
	e.state = engine.StateReady
	e.errorMessage = ""

	e.logger.Info("model loaded successfully (stub)", zap.String("model_path", modelPath))
	return nil
}

// Infer performs simulated inference
func (e *StubEngine) Infer(ctx context.Context, request engine.InferenceRequest) (*engine.InferenceResponse, error) {
	e.mu.Lock()
	if e.state != engine.StateReady && e.state != engine.StateIdle {
		e.mu.Unlock()
		return nil, fmt.Errorf("engine not ready: state=%s", e.state)
	}
	e.state = engine.StateInferring
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.state = engine.StateReady
		e.mu.Unlock()
	}()

	startTime := time.Now()

	// Simulate inference latency
	latency := time.Duration(e.simulateLatencyMs) * time.Millisecond
	// Add some randomness (±20%)
	jitter := time.Duration(rand.Intn(e.simulateLatencyMs/5)) * time.Millisecond
	if rand.Float64() < 0.5 {
		latency += jitter
	} else {
		latency -= jitter
	}
	time.Sleep(latency)

	// Simulate random inference failure if configured
	if e.simulateErrors && rand.Float64() < e.errorRate {
		return nil, fmt.Errorf("simulated inference error")
	}

	// Build prompt from request
	prompt := e.buildPrompt(request)

	// Generate simulated response
	responseText := e.generateResponse(prompt, request.Model)

	// Estimate tokens
	promptTokens := e.estimateTokens(prompt)
	completionTokens := e.estimateTokens(responseText)

	// Build response based on request type
	response := &engine.InferenceResponse{
		ID:      fmt.Sprintf("stub-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   request.Model,
		Choices: []engine.Choice{
			{
				Index: 0,
				Message: engine.Message{
					Role:    "assistant",
					Content: responseText,
				},
				Text:         responseText,
				FinishReason: "stop",
			},
		},
		Usage: engine.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}

	// Update statistics
	actualLatency := time.Since(startTime)
	e.mu.Lock()
	e.totalInferences++
	e.lastInferenceTime = time.Now()
	e.latencies = append(e.latencies, actualLatency)
	// Keep only last 100 latencies
	if len(e.latencies) > 100 {
		e.latencies = e.latencies[len(e.latencies)-100:]
	}
	e.mu.Unlock()

	e.logger.Info("inference completed (stub)",
		zap.Duration("latency", actualLatency),
		zap.Int("prompt_tokens", response.Usage.PromptTokens),
		zap.Int("completion_tokens", response.Usage.CompletionTokens),
	)

	return response, nil
}

// Unload simulates unloading the current model
func (e *StubEngine) Unload(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Info("unloading model (stub)", zap.String("model", e.loadedModel))
	e.state = engine.StateUnloading

	// Simulate unload delay
	time.Sleep(time.Duration(e.simulateLatencyMs/2) * time.Millisecond)

	e.loadedModel = ""
	e.state = engine.StateIdle

	return nil
}

// GetCapabilities returns the stub engine's capabilities
func (e *StubEngine) GetCapabilities() engine.EngineCapabilities {
	return engine.EngineCapabilities{
		Name:                   "stub",
		Version:                "1.0.0",
		Platform:               "test",
		SupportedFormats:       []string{"gguf", "safetensors", "pytorch"},
		SupportedQuantizations: []string{"q4", "q5", "q8", "fp16", "fp32"},
		MaxContextLength:       8192,
		SupportsStreaming:      false,
		SupportsChat:           true,
		SupportsCompletion:     true,
		GPUAcceleration:        false,
	}
}

// GetStatus returns the current engine status
func (e *StubEngine) GetStatus() engine.EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var avgLatency float64
	if len(e.latencies) > 0 {
		var sum time.Duration
		for _, lat := range e.latencies {
			sum += lat
		}
		avgLatency = float64(sum.Milliseconds()) / float64(len(e.latencies))
	}

	return engine.EngineStatus{
		State:             e.state,
		LoadedModel:       e.loadedModel,
		MemoryUsageMB:     100,
		LastInferenceTime: e.lastInferenceTime,
		TotalInferences:   e.totalInferences,
		AverageLatencyMs:  avgLatency,
		ErrorMessage:      e.errorMessage,
	}
}

// IsHealthy checks if the engine is healthy
func (e *StubEngine) IsHealthy() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.state == engine.StateReady || e.state == engine.StateIdle
}

// Helper methods

func (e *StubEngine) buildPrompt(request engine.InferenceRequest) string {
	if request.Prompt != "" {
		return request.Prompt
	}

	if len(request.Messages) == 0 {
		return ""
	}

	var prompt strings.Builder
	for _, msg := range request.Messages {
		prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	return prompt.String()
}

func (e *StubEngine) generateResponse(prompt, model string) string {
	responses := []string{
		"This is a simulated response from the local-stub inference engine.",
		"I'm a mock local LLM running in the local-stub engine for testing.",
		"This response demonstrates local inference routing in integration tests.",
		"The edge router successfully routed this request to the local-stub engine.",
		"Integration testing is working correctly with the local-stub engine.",
	}

	// Add some context based on prompt
	response := responses[rand.Intn(len(responses))]

	if strings.Contains(strings.ToLower(prompt), "hello") {
		response = "Hello! I'm the local-stub inference engine. How can I help you test the edge router?"
	} else if strings.Contains(strings.ToLower(prompt), "test") {
		response = "Test successful! The edge router is correctly routing requests to the local-stub engine."
	} else if strings.Contains(strings.ToLower(prompt), "local") {
		response = "Yes, this is running locally on the local-stub engine, not on the remote cluster."
	}

	return fmt.Sprintf("%s (Model: %s, Source: local-stub-engine)", response, model)
}

func (e *StubEngine) estimateTokens(text string) int {
	// Simple estimation: ~4 characters per token
	return len(text) / 4
}

// SetLatency allows changing the simulated latency for testing
func (e *StubEngine) SetLatency(latencyMs int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.simulateLatencyMs = latencyMs
}

// SetErrorRate allows changing the error rate for testing
func (e *StubEngine) SetErrorRate(rate float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errorRate = rate
	e.simulateErrors = rate > 0
}

// Made with Bob
