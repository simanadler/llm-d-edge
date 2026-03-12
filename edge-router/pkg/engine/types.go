package engine

import (
	"context"
	"time"
)

// InferenceEngine defines the interface that all platform-specific inference engines must implement
type InferenceEngine interface {
	// Initialize sets up the inference engine with the given configuration
	Initialize(ctx context.Context, config EngineConfig) error

	// LoadModel loads a model into memory
	LoadModel(ctx context.Context, modelPath string) error

	// Infer performs inference on the given request
	Infer(ctx context.Context, request InferenceRequest) (*InferenceResponse, error)

	// Unload unloads the current model from memory
	Unload(ctx context.Context) error

	// GetCapabilities returns the capabilities of this engine
	GetCapabilities() EngineCapabilities

	// GetStatus returns the current status of the engine
	GetStatus() EngineStatus

	// IsHealthy checks if the engine is healthy and ready to serve requests
	IsHealthy() bool
}

// EngineConfig contains configuration for an inference engine
type EngineConfig struct {
	// EngineName is the name of the engine (e.g., "mlx", "vllm", "llamacpp")
	EngineName string `json:"engine_name" yaml:"engine_name"`

	// ModelPath is the path to the model files
	ModelPath string `json:"model_path" yaml:"model_path"`

	// ModelFormat is the format of the model (e.g., "mlx", "gguf", "pytorch")
	ModelFormat string `json:"model_format" yaml:"model_format"`

	// Quantization is the quantization level (e.g., "4bit", "8bit", "none")
	Quantization string `json:"quantization" yaml:"quantization"`

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int `json:"max_tokens" yaml:"max_tokens"`

	// ContextLength is the maximum context length
	ContextLength int `json:"context_length" yaml:"context_length"`

	// GPULayers is the number of layers to offload to GPU (-1 for all)
	GPULayers int `json:"gpu_layers" yaml:"gpu_layers"`

	// NumThreads is the number of CPU threads to use
	NumThreads int `json:"num_threads" yaml:"num_threads"`

	// BatchSize is the batch size for inference
	BatchSize int `json:"batch_size" yaml:"batch_size"`

	// AdditionalArgs contains engine-specific additional arguments
	AdditionalArgs map[string]interface{} `json:"additional_args,omitempty" yaml:"additional_args,omitempty"`
}

// InferenceRequest represents a request for inference
type InferenceRequest struct {
	// Model is the model identifier
	Model string `json:"model"`

	// Messages contains the conversation messages (for chat completions)
	Messages []Message `json:"messages,omitempty"`

	// Prompt is the raw prompt (for completions)
	Prompt string `json:"prompt,omitempty"`

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0 to 2.0)
	Temperature float32 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling
	TopP float32 `json:"top_p,omitempty"`

	// TopK controls top-k sampling
	TopK int `json:"top_k,omitempty"`

	// Stream indicates whether to stream the response
	Stream bool `json:"stream,omitempty"`

	// Stop sequences
	Stop []string `json:"stop,omitempty"`

	// PresencePenalty penalizes new tokens based on presence
	PresencePenalty float32 `json:"presence_penalty,omitempty"`

	// FrequencyPenalty penalizes new tokens based on frequency
	FrequencyPenalty float32 `json:"frequency_penalty,omitempty"`

	// User identifier for tracking
	User string `json:"user,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // Message content
}

// InferenceResponse represents the response from inference
type InferenceResponse struct {
	// ID is a unique identifier for this response
	ID string `json:"id"`

	// Object type (e.g., "chat.completion", "text_completion")
	Object string `json:"object"`

	// Created timestamp
	Created int64 `json:"created"`

	// Model used for inference
	Model string `json:"model"`

	// Choices contains the generated completions
	Choices []Choice `json:"choices"`

	// Usage contains token usage information
	Usage Usage `json:"usage"`

	// Metadata contains additional information about the inference
	Metadata ResponseMetadata `json:"llm_d_metadata,omitempty"`
}

// Choice represents a single completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message,omitempty"`      // For chat completions
	Text         string  `json:"text,omitempty"`         // For text completions
	FinishReason string  `json:"finish_reason"`          // "stop", "length", "content_filter"
	Logprobs     *int    `json:"logprobs,omitempty"`     // Log probabilities
	Delta        *Message `json:"delta,omitempty"`       // For streaming
}

// Usage contains token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ResponseMetadata contains additional metadata about the inference
type ResponseMetadata struct {
	RoutingTarget   string  `json:"routing_target"`   // "local" or "remote"
	InferenceEngine string  `json:"inference_engine"` // Engine used
	LatencyMs       int64   `json:"latency_ms"`       // Total latency in milliseconds
	Platform        string  `json:"platform"`         // Platform (macos, windows, etc.)
	ModelFormat     string  `json:"model_format"`     // Model format used
	Quantization    string  `json:"quantization"`     // Quantization level
}

// EngineCapabilities describes what an engine can do
type EngineCapabilities struct {
	// Name of the engine
	Name string `json:"name"`

	// Version of the engine
	Version string `json:"version"`

	// SupportedFormats lists supported model formats
	SupportedFormats []string `json:"supported_formats"`

	// SupportedQuantizations lists supported quantization levels
	SupportedQuantizations []string `json:"supported_quantizations"`

	// MaxContextLength is the maximum context length supported
	MaxContextLength int `json:"max_context_length"`

	// SupportsStreaming indicates if streaming is supported
	SupportsStreaming bool `json:"supports_streaming"`

	// SupportsChat indicates if chat completions are supported
	SupportsChat bool `json:"supports_chat"`

	// SupportsCompletion indicates if text completions are supported
	SupportsCompletion bool `json:"supports_completion"`

	// GPUAcceleration indicates if GPU acceleration is available
	GPUAcceleration bool `json:"gpu_acceleration"`

	// Platform is the platform this engine runs on
	Platform string `json:"platform"`
}

// EngineStatus represents the current status of an engine
type EngineStatus struct {
	// State is the current state (idle, loading, ready, inferring, error)
	State EngineState `json:"state"`

	// LoadedModel is the currently loaded model (empty if none)
	LoadedModel string `json:"loaded_model"`

	// MemoryUsageMB is the current memory usage in megabytes
	MemoryUsageMB int64 `json:"memory_usage_mb"`

	// LastInferenceTime is the timestamp of the last inference
	LastInferenceTime time.Time `json:"last_inference_time"`

	// TotalInferences is the total number of inferences performed
	TotalInferences int64 `json:"total_inferences"`

	// AverageLatencyMs is the average inference latency in milliseconds
	AverageLatencyMs float64 `json:"average_latency_ms"`

	// ErrorMessage contains the last error message (if any)
	ErrorMessage string `json:"error_message,omitempty"`
}

// EngineState represents the state of an inference engine
type EngineState string

const (
	// StateIdle indicates the engine is idle
	StateIdle EngineState = "idle"

	// StateLoading indicates the engine is loading a model
	StateLoading EngineState = "loading"

	// StateReady indicates the engine is ready to serve requests
	StateReady EngineState = "ready"

	// StateInferring indicates the engine is currently performing inference
	StateInferring EngineState = "inferring"

	// StateError indicates the engine is in an error state
	StateError EngineState = "error"

	// StateUnloading indicates the engine is unloading a model
	StateUnloading EngineState = "unloading"
)

// EngineFactory is a function that creates a new inference engine
type EngineFactory func() InferenceEngine

// Registry holds registered engine factories
var engineRegistry = make(map[string]EngineFactory)

// RegisterEngine registers an engine factory with the given name
func RegisterEngine(name string, factory EngineFactory) {
	engineRegistry[name] = factory
}

// GetEngine returns an engine instance for the given name
func GetEngine(name string) (InferenceEngine, error) {
	factory, ok := engineRegistry[name]
	if !ok {
		return nil, ErrEngineNotFound
	}
	return factory(), nil
}

// GetRegisteredEngines returns a list of all registered engine names
func GetRegisteredEngines() []string {
	engines := make([]string, 0, len(engineRegistry))
	for name := range engineRegistry {
		engines = append(engines, name)
	}
	return engines
}

// Common errors
var (
	ErrEngineNotFound      = &EngineError{Code: "ENGINE_NOT_FOUND", Message: "inference engine not found"}
	ErrEngineNotInitialized = &EngineError{Code: "ENGINE_NOT_INITIALIZED", Message: "inference engine not initialized"}
	ErrModelNotLoaded      = &EngineError{Code: "MODEL_NOT_LOADED", Message: "model not loaded"}
	ErrInferenceTimeout    = &EngineError{Code: "INFERENCE_TIMEOUT", Message: "inference timeout"}
	ErrInvalidRequest      = &EngineError{Code: "INVALID_REQUEST", Message: "invalid inference request"}
)

// EngineError represents an engine-specific error
type EngineError struct {
	Code    string
	Message string
	Err     error
}

func (e *EngineError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *EngineError) Unwrap() error {
	return e.Err
}

// Made with Bob
