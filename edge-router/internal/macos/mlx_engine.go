// +build darwin

package macos

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
	"go.uber.org/zap"
)

// MLXEngine implements the InferenceEngine interface for macOS using MLX
type MLXEngine struct {
	mu     sync.RWMutex
	logger *zap.Logger

	config       engine.EngineConfig
	state        engine.EngineState
	loadedModel  string
	mlxProcess   *exec.Cmd
	errorMessage string

	// Statistics
	totalInferences   int64
	lastInferenceTime time.Time
	latencies         []time.Duration
}

// NewMLXEngine creates a new MLX inference engine
func NewMLXEngine(logger *zap.Logger) *MLXEngine {
	return &MLXEngine{
		logger:    logger,
		state:     engine.StateIdle,
		latencies: make([]time.Duration, 0, 100),
	}
}

// Initialize sets up the MLX engine
func (e *MLXEngine) Initialize(ctx context.Context, config engine.EngineConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Info("initializing MLX engine",
		zap.String("model_path", config.ModelPath),
		zap.String("model_format", config.ModelFormat),
		zap.String("quantization", config.Quantization),
	)

	// Store config first
	e.config = config

	// Verify MLX is installed
	if err := e.verifyMLXInstallation(); err != nil {
		e.state = engine.StateError
		e.errorMessage = err.Error()
		return fmt.Errorf("MLX not available: %w", err)
	}

	// Verify model exists
	if err := e.verifyModelExists(config.ModelPath); err != nil {
		e.state = engine.StateError
		e.errorMessage = err.Error()
		return fmt.Errorf("model not found: %w", err)
	}

	e.state = engine.StateIdle
	e.errorMessage = ""

	return nil
}

// LoadModel loads a model into memory
func (e *MLXEngine) LoadModel(ctx context.Context, modelPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Info("loading model", zap.String("model_path", modelPath))
	e.state = engine.StateLoading

	// Update config with new model path
	e.config.ModelPath = modelPath

	// MLX loads models on-demand during inference, so we just verify the model exists
	if err := e.verifyModelExists(modelPath); err != nil {
		e.state = engine.StateError
		e.errorMessage = err.Error()
		return fmt.Errorf("failed to load model: %w", err)
	}

	e.loadedModel = modelPath
	e.state = engine.StateReady
	e.errorMessage = ""

	e.logger.Info("model loaded successfully", zap.String("model_path", modelPath))
	return nil
}

// Infer performs inference using MLX
func (e *MLXEngine) Infer(ctx context.Context, request engine.InferenceRequest) (*engine.InferenceResponse, error) {
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

	// Build prompt from messages or use direct prompt
	prompt := e.buildPrompt(request)

	// Run MLX inference
	response, err := e.runMLXInference(ctx, prompt, request)
	if err != nil {
		e.mu.Lock()
		e.errorMessage = err.Error()
		e.mu.Unlock()
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	// Update statistics
	latency := time.Since(startTime)
	e.mu.Lock()
	e.totalInferences++
	e.lastInferenceTime = time.Now()
	e.latencies = append(e.latencies, latency)
	// Keep only last 100 latencies
	if len(e.latencies) > 100 {
		e.latencies = e.latencies[len(e.latencies)-100:]
	}
	e.mu.Unlock()

	e.logger.Info("inference completed",
		zap.Duration("latency", latency),
		zap.Int("prompt_tokens", response.Usage.PromptTokens),
		zap.Int("completion_tokens", response.Usage.CompletionTokens),
	)

	return response, nil
}

// Unload unloads the current model
func (e *MLXEngine) Unload(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Info("unloading model", zap.String("model", e.loadedModel))
	e.state = engine.StateUnloading

	// MLX doesn't require explicit unloading, but we reset state
	e.loadedModel = ""
	e.state = engine.StateIdle

	return nil
}

// GetCapabilities returns the engine's capabilities
func (e *MLXEngine) GetCapabilities() engine.EngineCapabilities {
	return engine.EngineCapabilities{
		Name:    "mlx",
		Version: "0.1.0",
		SupportedFormats: []string{
			"mlx",
			"gguf", // MLX can load GGUF via conversion
		},
		SupportedQuantizations: []string{
			"4bit",
			"8bit",
			"none",
		},
		MaxContextLength:  32768, // Depends on model
		SupportsStreaming: true,
		SupportsChat:      true,
		SupportsCompletion: true,
		GPUAcceleration:   true, // Metal acceleration
		Platform:          "macos",
	}
}

// GetStatus returns the current engine status
func (e *MLXEngine) GetStatus() engine.EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	avgLatency := 0.0
	if len(e.latencies) > 0 {
		sum := time.Duration(0)
		for _, l := range e.latencies {
			sum += l
		}
		avgLatency = float64(sum.Milliseconds()) / float64(len(e.latencies))
	}

	return engine.EngineStatus{
		State:             e.state,
		LoadedModel:       e.loadedModel,
		MemoryUsageMB:     e.estimateMemoryUsage(),
		LastInferenceTime: e.lastInferenceTime,
		TotalInferences:   e.totalInferences,
		AverageLatencyMs:  avgLatency,
		ErrorMessage:      e.errorMessage,
	}
}

// IsHealthy checks if the engine is healthy
func (e *MLXEngine) IsHealthy() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.state == engine.StateReady || e.state == engine.StateIdle
}

// Helper methods

func (e *MLXEngine) verifyMLXInstallation() error {
	// Check if mlx_lm is installed
	cmd := exec.Command("python3", "-c", "import mlx_lm")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mlx_lm not installed: %w", err)
	}
	return nil
}

func (e *MLXEngine) verifyModelExists(modelPath string) error {
	// Check if model directory exists
	cmd := exec.Command("test", "-d", modelPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("model directory not found: %s", modelPath)
	}
	return nil
}

func (e *MLXEngine) buildPrompt(request engine.InferenceRequest) string {
	if request.Prompt != "" {
		return request.Prompt
	}

	// Build prompt from messages
	var sb strings.Builder
	for _, msg := range request.Messages {
		switch msg.Role {
		case "system":
			sb.WriteString("System: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "user":
			sb.WriteString("User: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "assistant":
			sb.WriteString("Assistant: ")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}
	}
	sb.WriteString("Assistant: ")

	return sb.String()
}

func (e *MLXEngine) runMLXInference(ctx context.Context, prompt string, request engine.InferenceRequest) (*engine.InferenceResponse, error) {
	// Create a Python script to run MLX inference
	script := e.generateMLXScript(prompt, request)

	// Run the script directly with python3 -c
	cmd := exec.CommandContext(ctx, "python3", "-c", script)

	// Run the script
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("MLX inference failed: %w: %s", err, string(output))
	}

	// Parse the output (assuming JSON format)
	var result struct {
		Text          string `json:"text"`
		PromptTokens  int    `json:"prompt_tokens"`
		OutputTokens  int    `json:"output_tokens"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		// If JSON parsing fails, treat output as plain text
		result.Text = string(output)
		result.PromptTokens = len(prompt) / 4  // Rough estimate
		result.OutputTokens = len(result.Text) / 4
	}

	// Build response
	response := &engine.InferenceResponse{
		ID:      fmt.Sprintf("mlx-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   request.Model,
		Choices: []engine.Choice{
			{
				Index: 0,
				Message: engine.Message{
					Role:    "assistant",
					Content: result.Text,
				},
				FinishReason: "stop",
			},
		},
		Usage: engine.Usage{
			PromptTokens:     result.PromptTokens,
			CompletionTokens: result.OutputTokens,
			TotalTokens:      result.PromptTokens + result.OutputTokens,
		},
		Metadata: engine.ResponseMetadata{
			InferenceEngine: "mlx",
			ModelFormat:     e.config.ModelFormat,
			Quantization:    e.config.Quantization,
		},
	}

	return response, nil
}

func (e *MLXEngine) generateMLXScript(prompt string, request engine.InferenceRequest) string {
	maxTokens := request.MaxTokens
	if maxTokens == 0 {
		maxTokens = 100
	}

	temperature := request.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	// Generate Python script for MLX inference
	script := fmt.Sprintf(`
import json
from mlx_lm import load, generate

# Load model
model, tokenizer = load("%s")

# Generate
prompt = %s
output = generate(
    model,
    tokenizer,
    prompt=prompt,
    max_tokens=%d,
    temp=%.2f,
    verbose=False
)

# Calculate tokens (rough estimate)
prompt_tokens = len(tokenizer.encode(prompt))
output_tokens = len(tokenizer.encode(output))

# Output as JSON
result = {
    "text": output,
    "prompt_tokens": prompt_tokens,
    "output_tokens": output_tokens
}
print(json.dumps(result))
`, e.config.ModelPath, jsonQuote(prompt), maxTokens, temperature)

	return script
}

func (e *MLXEngine) estimateMemoryUsage() int64 {
	// Rough estimate based on model size
	// For a 7B model with 4-bit quantization: ~4GB
	// This is a placeholder - in production, query actual memory usage
	return 4096 // MB
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// Register the MLX engine
func init() {
	engine.RegisterEngine("mlx", func() engine.InferenceEngine {
		return NewMLXEngine(zap.NewNop())
	})
}

// Made with Bob
