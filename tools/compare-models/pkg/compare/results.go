package compare

import (
	"time"

	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
)

// ComparisonResult contains the complete comparison results
type ComparisonResult struct {
	Timestamp time.Time     `json:"timestamp"`
	Query     QueryInfo     `json:"query"`
	Models    []ModelResult `json:"models"`
	Summary   Summary       `json:"summary"`
}

// QueryInfo contains information about the query used for comparison
type QueryInfo struct {
	Messages    []engine.Message `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float32          `json:"temperature,omitempty"`
}

// ModelResult contains the result for a single model
type ModelResult struct {
	ModelName string             `json:"model_name"`
	ModelType string             `json:"model_type"` // "local" or "remote"
	Success   bool               `json:"success"`
	Error     string             `json:"error,omitempty"`
	Metrics   PerformanceMetrics `json:"metrics,omitempty"`
	Response  string             `json:"response,omitempty"`
}

// PerformanceMetrics contains performance measurements for a model
type PerformanceMetrics struct {
	TimeToFirstTokenMs int64   `json:"time_to_first_token_ms"`
	TotalLatencyMs     int64   `json:"total_latency_ms"`
	TokensPerSecond    float64 `json:"tokens_per_second"`
	PromptTokens       int     `json:"prompt_tokens"`
	CompletionTokens   int     `json:"completion_tokens"`
	TotalTokens        int     `json:"total_tokens"`
	MemoryUsageMB      int64   `json:"memory_usage_mb,omitempty"`
	FinishReason       string  `json:"finish_reason"`
}

// Summary contains aggregate statistics across all models
type Summary struct {
	TotalModels      int     `json:"total_models"`
	Successful       int     `json:"successful"`
	Failed           int     `json:"failed"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	FastestModel     string  `json:"fastest_model"`
	SlowestModel     string  `json:"slowest_model"`
	HighestTPSModel  string  `json:"highest_tps_model"`
}

// Made with Bob