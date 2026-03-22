package compare

import (
	"time"

	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
)

// calculateMetrics computes performance metrics from an inference response
func calculateMetrics(resp *engine.InferenceResponse, latency time.Duration, status engine.EngineStatus) PerformanceMetrics {
	// Extract usage data from response (reused from edge-router)
	usage := resp.Usage

	// Calculate tokens per second
	var tps float64
	if usage.CompletionTokens > 0 && latency.Seconds() > 0 {
		tps = float64(usage.CompletionTokens) / latency.Seconds()
	}

	// For non-streaming, TTFT equals total latency
	// For streaming, this would be measured separately
	ttft := latency

	metrics := PerformanceMetrics{
		TimeToFirstTokenMs: ttft.Milliseconds(),
		TotalLatencyMs:     latency.Milliseconds(),
		TokensPerSecond:    tps,
		PromptTokens:       usage.PromptTokens,
		CompletionTokens:   usage.CompletionTokens,
		TotalTokens:        usage.TotalTokens,
		FinishReason:       extractFinishReason(resp),
	}

	// Add memory usage if available (local models only)
	if status.MemoryUsageMB > 0 {
		metrics.MemoryUsageMB = status.MemoryUsageMB
	}

	return metrics
}

// extractFinishReason extracts the finish reason from the response
func extractFinishReason(resp *engine.InferenceResponse) string {
	if len(resp.Choices) > 0 {
		return resp.Choices[0].FinishReason
	}
	return "unknown"
}

// extractResponse extracts the generated text from the response
func extractResponse(resp *engine.InferenceResponse) string {
	if len(resp.Choices) > 0 {
		// For chat completions
		if resp.Choices[0].Message.Content != "" {
			return resp.Choices[0].Message.Content
		}
		// For text completions
		return resp.Choices[0].Text
	}
	return ""
}

// extractQueryInfo creates QueryInfo from an inference request
func extractQueryInfo(req *engine.InferenceRequest) QueryInfo {
	return QueryInfo{
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
}

// Made with Bob