package compare

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/llm-d-incubation/llm-d-edge/pkg/config"
	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
	"github.com/llm-d-incubation/llm-d-edge/pkg/router"
	"go.uber.org/zap"
)

// Comparator orchestrates the comparison of multiple models
type Comparator struct {
	config  *config.Config
	logger  *zap.Logger
	results []ModelResult
}

// NewComparator creates a new Comparator instance
func NewComparator(cfg *config.Config, logger *zap.Logger) *Comparator {
	return &Comparator{
		config:  cfg,
		logger:  logger,
		results: make([]ModelResult, 0),
	}
}

// RunComparison executes the comparison across all models
func (c *Comparator) RunComparison(ctx context.Context, req *engine.InferenceRequest, remoteModel string) (*ComparisonResult, error) {
	c.logger.Info("starting model comparison",
		zap.Int("local_models", len(c.config.Edge.Models.Local)),
		zap.String("remote_model", remoteModel),
	)

	// Compare all local models
	for i, localModel := range c.config.Edge.Models.Local {
		c.logger.Info("comparing local model",
			zap.Int("index", i+1),
			zap.Int("total", len(c.config.Edge.Models.Local)),
			zap.String("model", localModel.Name),
		)

		result := c.compareLocalModel(ctx, localModel, req)
		c.results = append(c.results, result)

		// Log result
		if result.Success {
			c.logger.Info("local model comparison succeeded",
				zap.String("model", localModel.Name),
				zap.Int64("latency_ms", result.Metrics.TotalLatencyMs),
				zap.Float64("tokens_per_sec", result.Metrics.TokensPerSecond),
			)
		} else {
			c.logger.Warn("local model comparison failed",
				zap.String("model", localModel.Name),
				zap.String("error", result.Error),
			)
		}
	}

	// Compare remote model
	c.logger.Info("comparing remote model", zap.String("model", remoteModel))
	remoteResult := c.compareRemoteModel(ctx, remoteModel, req)
	c.results = append(c.results, remoteResult)

	if remoteResult.Success {
		c.logger.Info("remote model comparison succeeded",
			zap.String("model", remoteModel),
			zap.Int64("latency_ms", remoteResult.Metrics.TotalLatencyMs),
			zap.Float64("tokens_per_sec", remoteResult.Metrics.TokensPerSecond),
		)
	} else {
		c.logger.Warn("remote model comparison failed",
			zap.String("model", remoteModel),
			zap.String("error", remoteResult.Error),
		)
	}

	// Generate summary
	summary := c.generateSummary()

	c.logger.Info("comparison complete",
		zap.Int("total", summary.TotalModels),
		zap.Int("successful", summary.Successful),
		zap.Int("failed", summary.Failed),
	)

	return &ComparisonResult{
		Timestamp: time.Now(),
		Query:     extractQueryInfo(req),
		Models:    c.results,
		Summary:   summary,
	}, nil
}

// compareLocalModel compares a single local model
func (c *Comparator) compareLocalModel(ctx context.Context, model config.ExtendedLocalModelConfig, req *engine.InferenceRequest) ModelResult {
	// Get the appropriate engine for the platform (reused from edge-router)
	engineName := getEngineForPlatform(c.config.Edge.Platform)
	eng, err := engine.GetEngine(engineName)
	if err != nil {
		return ModelResult{
			ModelName: model.Name,
			ModelType: "local",
			Success:   false,
			Error:     fmt.Sprintf("failed to get engine: %v", err),
		}
	}

	// Initialize engine (reused pattern from edge-router)
	engineConfig := engine.EngineConfig{
		EngineName:    engineName,
		ModelPath:     model.Path,
		ModelFormat:   model.Format,
		Quantization:  model.Quantization,
		MaxTokens:     2048,
		ContextLength: 4096,
		GPULayers:     -1, // Use all available
	}

	if err := eng.Initialize(ctx, engineConfig); err != nil {
		return ModelResult{
			ModelName: model.Name,
			ModelType: "local",
			Success:   false,
			Error:     fmt.Sprintf("initialization failed: %v", err),
		}
	}

	// Load model
	if err := eng.LoadModel(ctx, model.Path); err != nil {
		return ModelResult{
			ModelName: model.Name,
			ModelType: "local",
			Success:   false,
			Error:     fmt.Sprintf("model loading failed: %v", err),
		}
	}

	// Measure inference
	startTime := time.Now()
	resp, err := eng.Infer(ctx, *req)
	totalLatency := time.Since(startTime)

	// Unload model to free memory
	if unloadErr := eng.Unload(ctx); unloadErr != nil {
		c.logger.Warn("failed to unload model", zap.String("model", model.Name), zap.Error(unloadErr))
	}

	if err != nil {
		return ModelResult{
			ModelName: model.Name,
			ModelType: "local",
			Success:   false,
			Error:     fmt.Sprintf("inference failed: %v", err),
		}
	}

	// Calculate metrics
	metrics := calculateMetrics(resp, totalLatency, eng.GetStatus())

	return ModelResult{
		ModelName: model.Name,
		ModelType: "local",
		Success:   true,
		Metrics:   metrics,
		Response:  extractResponse(resp),
	}
}

// compareRemoteModel compares a remote model
func (c *Comparator) compareRemoteModel(ctx context.Context, modelName string, req *engine.InferenceRequest) ModelResult {
	// Create remote client (reused from edge-router)
	client, err := router.NewRemoteClient(c.config.Edge.Models.Remote, c.logger)
	if err != nil {
		return ModelResult{
			ModelName: modelName,
			ModelType: "remote",
			Success:   false,
			Error:     fmt.Sprintf("failed to create remote client: %v", err),
		}
	}

	// Update request with remote model name
	remoteReq := *req
	remoteReq.Model = modelName

	// Measure inference
	startTime := time.Now()
	resp, err := client.Infer(ctx, &remoteReq)
	totalLatency := time.Since(startTime)

	if err != nil {
		return ModelResult{
			ModelName: modelName,
			ModelType: "remote",
			Success:   false,
			Error:     fmt.Sprintf("inference failed: %v", err),
		}
	}

	// Calculate metrics (no memory stats for remote)
	metrics := calculateMetrics(resp, totalLatency, engine.EngineStatus{})

	return ModelResult{
		ModelName: modelName,
		ModelType: "remote",
		Success:   true,
		Metrics:   metrics,
		Response:  extractResponse(resp),
	}
}

// generateSummary creates aggregate statistics from all results
func (c *Comparator) generateSummary() Summary {
	summary := Summary{
		TotalModels: len(c.results),
	}

	var totalLatency int64
	var fastestLatency int64 = -1
	var slowestLatency int64
	var highestTPS float64

	for _, result := range c.results {
		if result.Success {
			summary.Successful++
			latency := result.Metrics.TotalLatencyMs
			tps := result.Metrics.TokensPerSecond

			totalLatency += latency

			// Track fastest
			if fastestLatency == -1 || latency < fastestLatency {
				fastestLatency = latency
				summary.FastestModel = result.ModelName
			}

			// Track slowest
			if latency > slowestLatency {
				slowestLatency = latency
				summary.SlowestModel = result.ModelName
			}

			// Track highest TPS
			if tps > highestTPS {
				highestTPS = tps
				summary.HighestTPSModel = result.ModelName
			}
		} else {
			summary.Failed++
		}
	}

	// Calculate average latency
	if summary.Successful > 0 {
		summary.AverageLatencyMs = float64(totalLatency) / float64(summary.Successful)
	}

	return summary
}

// getEngineForPlatform returns the appropriate engine name for the platform
// Reused from edge-router/cmd/edge-router/main.go
func getEngineForPlatform(platform string) string {
	switch platform {
	case "stub", "test", "local-stub":
		return "stub"
	case "macos":
		return "mlx"
	case "windows":
		return "vllm" // or "llamacpp"
	case "android":
		return "llamacpp"
	case "ios":
		return "coreml"
	default:
		// Auto-detect based on runtime
		switch runtime.GOOS {
		case "darwin":
			return "mlx"
		case "windows":
			return "vllm"
		default:
			return "llamacpp"
		}
	}
}

// Made with Bob