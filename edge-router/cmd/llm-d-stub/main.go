// llm-d Stub Server
// A mock implementation of the llm-d Kubernetes cluster for integration testing
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	port           = flag.Int("port", 8081, "Port to listen on")
	latencyMin     = flag.Int("latency-min", 100, "Minimum latency in milliseconds")
	latencyMax     = flag.Int("latency-max", 500, "Maximum latency in milliseconds")
	errorRate      = flag.Float64("error-rate", 0.0, "Error rate (0.0-1.0)")
	requireAPIKey  = flag.Bool("require-api-key", false, "Require API key authentication")
	validAPIKey    = flag.String("api-key", "test-api-key", "Valid API key for authentication")
	logRequests    = flag.Bool("log-requests", true, "Log all requests")
)

// OpenAI-compatible request/response types
type ChatCompletionRequest struct {
	Model       string                 `json:"model"`
	Messages    []ChatMessage          `json:"messages"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
	N           int                    `json:"n,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []ChatCompletionChoice   `json:"choices"`
	Usage   ChatCompletionUsage      `json:"usage"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type CompletionRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Temperature float64  `json:"temperature,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
	Stop        []string `json:"stop,omitempty"`
	N           int      `json:"n,omitempty"`
	TopP        float64  `json:"top_p,omitempty"`
}

type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   ChatCompletionUsage `json:"usage"`
}

type CompletionChoice struct {
	Text         string `json:"text"`
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

func main() {
	flag.Parse()

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Set Gin mode
	if !*logRequests {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Middleware for API key authentication
	if *requireAPIKey {
		r.Use(authMiddleware())
	}

	// Middleware for simulating latency
	r.Use(latencyMiddleware())

	// Middleware for simulating errors
	r.Use(errorMiddleware())

	// OpenAI-compatible endpoints
	r.POST("/v1/chat/completions", handleChatCompletion)
	r.POST("/v1/completions", handleCompletion)
	
	// Health and metrics endpoints
	r.GET("/health", handleHealth)
	r.GET("/metrics", handleMetrics)
	r.GET("/ready", handleReady)

	// Info endpoint
	r.GET("/", handleInfo)

	log.Printf("llm-d Stub Server starting on port %d", *port)
	log.Printf("Configuration:")
	log.Printf("  - Latency: %d-%d ms", *latencyMin, *latencyMax)
	log.Printf("  - Error rate: %.1f%%", *errorRate*100)
	log.Printf("  - API key required: %v", *requireAPIKey)
	log.Printf("  - Log requests: %v", *logRequests)

	if err := r.Run(fmt.Sprintf(":%d", *port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: ErrorDetail{
					Message: "Missing Authorization header",
					Type:    "authentication_error",
				},
			})
			c.Abort()
			return
		}

		// Check Bearer token
		if !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: ErrorDetail{
					Message: "Invalid Authorization header format",
					Type:    "authentication_error",
				},
			})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token != *validAPIKey {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: ErrorDetail{
					Message: "Invalid API key",
					Type:    "authentication_error",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func latencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip latency for health/metrics endpoints
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/metrics" || c.Request.URL.Path == "/ready" {
			c.Next()
			return
		}

		// Simulate latency
		latency := *latencyMin + rand.Intn(*latencyMax-*latencyMin+1)
		time.Sleep(time.Duration(latency) * time.Millisecond)
		c.Next()
	}
}

func errorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip errors for health/metrics endpoints
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/metrics" || c.Request.URL.Path == "/ready" {
			c.Next()
			return
		}

		// Simulate random errors
		if rand.Float64() < *errorRate {
			errorTypes := []struct {
				status  int
				message string
				errType string
			}{
				{http.StatusServiceUnavailable, "Service temporarily unavailable", "service_error"},
				{http.StatusTooManyRequests, "Rate limit exceeded", "rate_limit_error"},
				{http.StatusInternalServerError, "Internal server error", "server_error"},
				{http.StatusBadGateway, "Bad gateway", "gateway_error"},
			}
			
			errInfo := errorTypes[rand.Intn(len(errorTypes))]
			c.JSON(errInfo.status, ErrorResponse{
				Error: ErrorDetail{
					Message: errInfo.message,
					Type:    errInfo.errType,
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func handleChatCompletion(c *gin.Context) {
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Message: fmt.Sprintf("Invalid request: %v", err),
				Type:    "invalid_request_error",
			},
		})
		return
	}

	if *logRequests {
		log.Printf("Chat completion request: model=%s, messages=%d", req.Model, len(req.Messages))
	}

	// Generate response
	response := generateChatResponse(req)
	c.JSON(http.StatusOK, response)
}

func handleCompletion(c *gin.Context) {
	var req CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Message: fmt.Sprintf("Invalid request: %v", err),
				Type:    "invalid_request_error",
			},
		})
		return
	}

	if *logRequests {
		log.Printf("Completion request: model=%s, prompt_length=%d", req.Model, len(req.Prompt))
	}

	// Generate response
	response := generateCompletionResponse(req)
	c.JSON(http.StatusOK, response)
}

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "llm-d-stub",
		"timestamp": time.Now().Unix(),
	})
}

func handleReady(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ready": true,
		"service": "llm-d-stub",
	})
}

func handleMetrics(c *gin.Context) {
	// Simple Prometheus-style metrics
	metrics := `# HELP llm_d_stub_requests_total Total number of requests
# TYPE llm_d_stub_requests_total counter
llm_d_stub_requests_total 0

# HELP llm_d_stub_latency_seconds Request latency in seconds
# TYPE llm_d_stub_latency_seconds histogram
llm_d_stub_latency_seconds_bucket{le="0.1"} 0
llm_d_stub_latency_seconds_bucket{le="0.5"} 0
llm_d_stub_latency_seconds_bucket{le="1.0"} 0
llm_d_stub_latency_seconds_bucket{le="+Inf"} 0
llm_d_stub_latency_seconds_sum 0
llm_d_stub_latency_seconds_count 0
`
	c.String(http.StatusOK, metrics)
}

func handleInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "llm-d-stub",
		"version": "1.0.0",
		"description": "Mock llm-d Kubernetes cluster for integration testing",
		"endpoints": []string{
			"/v1/chat/completions",
			"/v1/completions",
			"/health",
			"/metrics",
			"/ready",
		},
		"configuration": gin.H{
			"latency_range_ms": fmt.Sprintf("%d-%d", *latencyMin, *latencyMax),
			"error_rate": *errorRate,
			"api_key_required": *requireAPIKey,
		},
	})
}

func generateChatResponse(req ChatCompletionRequest) ChatCompletionResponse {
	// Extract last user message
	var lastMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastMessage = req.Messages[i].Content
			break
		}
	}

	// Generate simple response
	responseText := generateResponseText(lastMessage, req.Model)

	// Estimate tokens
	promptTokens := estimateTokens(lastMessage)
	completionTokens := estimateTokens(responseText)

	return ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-stub-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []ChatCompletionChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: responseText,
				},
				FinishReason: "stop",
			},
		},
		Usage: ChatCompletionUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}

func generateCompletionResponse(req CompletionRequest) CompletionResponse {
	responseText := generateResponseText(req.Prompt, req.Model)

	promptTokens := estimateTokens(req.Prompt)
	completionTokens := estimateTokens(responseText)

	return CompletionResponse{
		ID:      fmt.Sprintf("cmpl-stub-%d", time.Now().UnixNano()),
		Object:  "text_completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []CompletionChoice{
			{
				Text:         responseText,
				Index:        0,
				FinishReason: "stop",
			},
		},
		Usage: ChatCompletionUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}
}

func generateResponseText(prompt, model string) string {
	responses := []string{
		"This is a simulated response from the llm-d stub server.",
		"I'm a mock LLM running in the stub Kubernetes cluster.",
		"This response demonstrates remote inference routing.",
		"The edge router successfully routed this request to the remote cluster.",
		"Integration testing is working correctly with the stub server.",
	}

	// Add some context based on prompt
	response := responses[rand.Intn(len(responses))]
	
	if strings.Contains(strings.ToLower(prompt), "hello") {
		response = "Hello! I'm the llm-d stub server. How can I help you test the edge router?"
	} else if strings.Contains(strings.ToLower(prompt), "test") {
		response = "Test successful! The edge router is correctly routing requests to the remote llm-d cluster (stub)."
	}

	return fmt.Sprintf("%s (Model: %s, Source: llm-d-stub)", response, model)
}

func estimateTokens(text string) int {
	// Simple estimation: ~4 characters per token
	return len(text) / 4
}

// Made with Bob
