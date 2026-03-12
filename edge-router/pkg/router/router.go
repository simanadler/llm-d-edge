package router

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/llm-d-incubation/llm-d-edge/pkg/config"
	"github.com/llm-d-incubation/llm-d-edge/pkg/engine"
	"go.uber.org/zap"
)

// Router handles routing decisions between local and remote inference
type Router struct {
	config        *config.Config
	localEngine   engine.InferenceEngine
	remoteClient  *RemoteClient
	logger        *zap.Logger
	metrics       *Metrics
	networkStatus *NetworkStatus
}

// NewRouter creates a new router instance
func NewRouter(cfg *config.Config, localEngine engine.InferenceEngine, logger *zap.Logger) (*Router, error) {
	remoteClient, err := NewRemoteClient(cfg.Edge.Models.Remote, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote client: %w", err)
	}

	return &Router{
		config:        cfg,
		localEngine:   localEngine,
		remoteClient:  remoteClient,
		logger:        logger,
		metrics:       NewMetrics(),
		networkStatus: NewNetworkStatus(),
	}, nil
}

// RouteTarget represents where a request should be routed
type RouteTarget string

const (
	TargetLocal  RouteTarget = "local"
	TargetRemote RouteTarget = "remote"
)

// RoutingDecision contains the routing decision and metadata
type RoutingDecision struct {
	Target      RouteTarget
	Reason      string
	Confidence  float64 // 0.0 to 1.0
	EstimatedMs int64   // Estimated latency in milliseconds
}

// Route determines where to route an inference request
func (r *Router) Route(ctx context.Context, req *engine.InferenceRequest) (*RoutingDecision, error) {
	startTime := time.Now()
	defer func() {
		r.metrics.RecordRoutingDecision(time.Since(startTime))
	}()

	// Check if model is available locally
	hasLocalModel := r.hasLocalModel(req.Model)

	// Check network connectivity
	isConnected := r.networkStatus.IsConnected()

	// Apply routing policy
	decision, err := r.applyRoutingPolicy(req, hasLocalModel, isConnected)
	if err != nil {
		return nil, fmt.Errorf("failed to apply routing policy: %w", err)
	}

	// Apply routing rules (can override policy)
	if overrideDecision := r.applyRoutingRules(req, hasLocalModel, isConnected); overrideDecision != nil {
		decision = overrideDecision
	}

	r.logger.Info("routing decision",
		zap.String("target", string(decision.Target)),
		zap.String("reason", decision.Reason),
		zap.Float64("confidence", decision.Confidence),
		zap.String("model", req.Model),
	)

	return decision, nil
}

// applyRoutingPolicy applies the configured routing policy
func (r *Router) applyRoutingPolicy(req *engine.InferenceRequest, hasLocalModel, isConnected bool) (*RoutingDecision, error) {
	policy := r.config.Edge.Routing.Policy

	// If model not available locally, must route to remote (if connected)
	if !hasLocalModel {
		if !isConnected {
			return nil, fmt.Errorf("model not available locally and no network connection")
		}
		return &RoutingDecision{
			Target:     TargetRemote,
			Reason:     "model_not_available_locally",
			Confidence: 1.0,
		}, nil
	}

	// If not connected, must use local
	if !isConnected {
		return &RoutingDecision{
			Target:     TargetLocal,
			Reason:     "network_offline",
			Confidence: 1.0,
		}, nil
	}

	// Apply policy-specific logic
	switch policy {
	case config.PolicyLocalFirst:
		return r.applyLocalFirstPolicy(req)

	case config.PolicyRemoteFirst:
		return r.applyRemoteFirstPolicy(req)

	case config.PolicyHybrid:
		return r.applyHybridPolicy(req)

	case config.PolicyCostOptimized:
		return r.applyCostOptimizedPolicy(req)

	case config.PolicyLatencyOptimized:
		return r.applyLatencyOptimizedPolicy(req)

	case config.PolicyMobileOptimized:
		return r.applyMobileOptimizedPolicy(req)

	default:
		return nil, fmt.Errorf("unknown routing policy: %s", policy)
	}
}

// applyLocalFirstPolicy prefers local inference when possible
func (r *Router) applyLocalFirstPolicy(req *engine.InferenceRequest) (*RoutingDecision, error) {
	// Check if local engine can handle the request
	if r.canServeLocally(req) {
		return &RoutingDecision{
			Target:      TargetLocal,
			Reason:      "local_first_policy",
			Confidence:  0.9,
			EstimatedMs: r.estimateLocalLatency(req),
		}, nil
	}

	return &RoutingDecision{
		Target:      TargetRemote,
		Reason:      "local_capacity_exceeded",
		Confidence:  0.8,
		EstimatedMs: r.estimateRemoteLatency(req),
	}, nil
}

// applyRemoteFirstPolicy prefers remote inference
func (r *Router) applyRemoteFirstPolicy(req *engine.InferenceRequest) (*RoutingDecision, error) {
	return &RoutingDecision{
		Target:      TargetRemote,
		Reason:      "remote_first_policy",
		Confidence:  0.9,
		EstimatedMs: r.estimateRemoteLatency(req),
	}, nil
}

// applyHybridPolicy balances between local and remote
func (r *Router) applyHybridPolicy(req *engine.InferenceRequest) (*RoutingDecision, error) {
	promptTokens := r.estimatePromptTokens(req)

	// Use local for small requests, remote for large
	if promptTokens < 1000 && r.canServeLocally(req) {
		return &RoutingDecision{
			Target:      TargetLocal,
			Reason:      "hybrid_policy_small_request",
			Confidence:  0.85,
			EstimatedMs: r.estimateLocalLatency(req),
		}, nil
	}

	return &RoutingDecision{
		Target:      TargetRemote,
		Reason:      "hybrid_policy_large_request",
		Confidence:  0.85,
		EstimatedMs: r.estimateRemoteLatency(req),
	}, nil
}

// applyCostOptimizedPolicy minimizes cost
func (r *Router) applyCostOptimizedPolicy(req *engine.InferenceRequest) (*RoutingDecision, error) {
	localCost := r.estimateLocalCost(req)
	remoteCost := r.estimateRemoteCost(req)

	if localCost < remoteCost && r.canServeLocally(req) {
		return &RoutingDecision{
			Target:      TargetLocal,
			Reason:      "cost_optimized_local_cheaper",
			Confidence:  0.8,
			EstimatedMs: r.estimateLocalLatency(req),
		}, nil
	}

	return &RoutingDecision{
		Target:      TargetRemote,
		Reason:      "cost_optimized_remote_cheaper",
		Confidence:  0.8,
		EstimatedMs: r.estimateRemoteLatency(req),
	}, nil
}

// applyLatencyOptimizedPolicy minimizes latency
func (r *Router) applyLatencyOptimizedPolicy(req *engine.InferenceRequest) (*RoutingDecision, error) {
	// For streaming requests with small prompts, prefer local
	if req.Stream && r.estimatePromptTokens(req) < 2000 {
		if r.canServeLocally(req) {
			return &RoutingDecision{
				Target:      TargetLocal,
				Reason:      "latency_optimized_interactive",
				Confidence:  0.9,
				EstimatedMs: r.estimateLocalLatency(req),
			}, nil
		}
	}

	// For batch requests, use remote (better throughput)
	return &RoutingDecision{
		Target:      TargetRemote,
		Reason:      "latency_optimized_batch",
		Confidence:  0.85,
		EstimatedMs: r.estimateRemoteLatency(req),
	}, nil
}

// applyMobileOptimizedPolicy optimizes for mobile constraints
func (r *Router) applyMobileOptimizedPolicy(req *engine.InferenceRequest) (*RoutingDecision, error) {
	// Check battery level
	if r.isBatteryLow() {
		return &RoutingDecision{
			Target:     TargetRemote,
			Reason:     "mobile_optimized_battery_low",
			Confidence: 0.95,
		}, nil
	}

	// Check thermal state
	if r.isThermalThrottling() {
		return &RoutingDecision{
			Target:     TargetRemote,
			Reason:     "mobile_optimized_thermal_throttling",
			Confidence: 0.95,
		}, nil
	}

	// For small requests, use local
	if r.estimatePromptTokens(req) < 500 && r.canServeLocally(req) {
		return &RoutingDecision{
			Target:      TargetLocal,
			Reason:      "mobile_optimized_small_request",
			Confidence:  0.8,
			EstimatedMs: r.estimateLocalLatency(req),
		}, nil
	}

	// Default to remote for mobile
	return &RoutingDecision{
		Target:      TargetRemote,
		Reason:      "mobile_optimized_default",
		Confidence:  0.75,
		EstimatedMs: r.estimateRemoteLatency(req),
	}, nil
}

// applyRoutingRules applies custom routing rules
func (r *Router) applyRoutingRules(req *engine.InferenceRequest, hasLocalModel, isConnected bool) *RoutingDecision {
	for _, rule := range r.config.Edge.RoutingRules {
		if r.evaluateCondition(rule.Condition, req, hasLocalModel, isConnected) {
			return r.executeAction(rule.Action, rule.Condition)
		}
	}
	return nil
}

// evaluateCondition evaluates a routing rule condition
func (r *Router) evaluateCondition(condition string, req *engine.InferenceRequest, hasLocalModel, isConnected bool) bool {
	// Simple expression evaluator (in production, use a proper expression parser)
	condition = strings.ToLower(condition)

	// Check for common conditions
	if strings.Contains(condition, "network_offline") && !isConnected {
		return true
	}

	if strings.Contains(condition, "model in local_models") && hasLocalModel {
		return true
	}

	promptTokens := r.estimatePromptTokens(req)
	if strings.Contains(condition, "prompt_tokens < 1000") && promptTokens < 1000 {
		return true
	}

	if strings.Contains(condition, "prompt_tokens >= 1000") && promptTokens >= 1000 {
		return true
	}

	return false
}

// executeAction executes a routing rule action
func (r *Router) executeAction(action, condition string) *RoutingDecision {
	switch action {
	case "route_local":
		return &RoutingDecision{
			Target:     TargetLocal,
			Reason:     fmt.Sprintf("routing_rule: %s", condition),
			Confidence: 1.0,
		}
	case "route_remote":
		return &RoutingDecision{
			Target:     TargetRemote,
			Reason:     fmt.Sprintf("routing_rule: %s", condition),
			Confidence: 1.0,
		}
	case "route_local_or_fail":
		return &RoutingDecision{
			Target:     TargetLocal,
			Reason:     fmt.Sprintf("routing_rule: %s", condition),
			Confidence: 1.0,
		}
	default:
		return nil
	}
}

// Helper methods

func (r *Router) hasLocalModel(modelName string) bool {
	for _, model := range r.config.Edge.Models.Local {
		if model.Name == modelName {
			return true
		}
	}
	return false
}

func (r *Router) canServeLocally(req *engine.InferenceRequest) bool {
	if !r.localEngine.IsHealthy() {
		return false
	}

	status := r.localEngine.GetStatus()
	if status.State != engine.StateReady && status.State != engine.StateIdle {
		return false
	}

	// Check if we have enough memory (simplified check)
	if status.MemoryUsageMB > 32000 { // Arbitrary threshold
		return false
	}

	return true
}

func (r *Router) estimatePromptTokens(req *engine.InferenceRequest) int {
	// Simple estimation: ~4 characters per token
	if req.Prompt != "" {
		return len(req.Prompt) / 4
	}

	totalChars := 0
	for _, msg := range req.Messages {
		totalChars += len(msg.Content)
	}
	return totalChars / 4
}

func (r *Router) estimateLocalLatency(req *engine.InferenceRequest) int64 {
	// Simplified estimation based on prompt tokens
	promptTokens := r.estimatePromptTokens(req)
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 100
	}

	// Rough estimate: 50ms per token for generation
	return int64(promptTokens*10 + maxTokens*50)
}

func (r *Router) estimateRemoteLatency(req *engine.InferenceRequest) int64 {
	// Add network overhead
	return r.estimateLocalLatency(req) + 200 // 200ms network overhead
}

func (r *Router) estimateLocalCost(req *engine.InferenceRequest) float64 {
	// Local inference is essentially free (just electricity)
	return 0.0001
}

func (r *Router) estimateRemoteCost(req *engine.InferenceRequest) float64 {
	// Estimate based on tokens (simplified)
	promptTokens := float64(r.estimatePromptTokens(req))
	completionTokens := float64(req.MaxTokens)
	if completionTokens == 0 {
		completionTokens = 100
	}

	// Example pricing: $0.01 per 1K tokens
	return (promptTokens + completionTokens) / 1000.0 * 0.01
}

func (r *Router) isBatteryLow() bool {
	// Platform-specific implementation needed
	// For now, return false
	return false
}

func (r *Router) isThermalThrottling() bool {
	// Platform-specific implementation needed
	// For now, return false
	return false
}

// Infer performs inference using the appropriate target
func (r *Router) Infer(ctx context.Context, req *engine.InferenceRequest) (*engine.InferenceResponse, error) {
	// Make routing decision
	decision, err := r.Route(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	startTime := time.Now()
	var response *engine.InferenceResponse

	// Execute inference based on decision
	switch decision.Target {
	case TargetLocal:
		response, err = r.localEngine.Infer(ctx, *req)
		if err != nil {
			// Try fallback if configured
			if r.config.Edge.Routing.Fallback == "remote" && r.networkStatus.IsConnected() {
				r.logger.Warn("local inference failed, falling back to remote", zap.Error(err))
				response, err = r.remoteClient.Infer(ctx, req)
				if err != nil {
					return nil, fmt.Errorf("local and remote inference failed: %w", err)
				}
				decision.Target = TargetRemote
			} else {
				return nil, fmt.Errorf("local inference failed: %w", err)
			}
		}

	case TargetRemote:
		response, err = r.remoteClient.Infer(ctx, req)
		if err != nil {
			// Try fallback if configured
			if r.config.Edge.Routing.Fallback == "local" && r.hasLocalModel(req.Model) {
				r.logger.Warn("remote inference failed, falling back to local", zap.Error(err))
				response, err = r.localEngine.Infer(ctx, *req)
				if err != nil {
					return nil, fmt.Errorf("remote and local inference failed: %w", err)
				}
				decision.Target = TargetLocal
			} else {
				return nil, fmt.Errorf("remote inference failed: %w", err)
			}
		}
	}

	// Add metadata to response
	latencyMs := time.Since(startTime).Milliseconds()
	response.Metadata = engine.ResponseMetadata{
		RoutingTarget:   string(decision.Target),
		InferenceEngine: r.getEngineNameForTarget(decision.Target),
		LatencyMs:       latencyMs,
		Platform:        r.config.Edge.Platform,
	}

	// Record metrics
	r.metrics.RecordInference(decision.Target, latencyMs, err == nil)

	return response, nil
}

func (r *Router) getEngineNameForTarget(target RouteTarget) string {
	if target == TargetLocal {
		caps := r.localEngine.GetCapabilities()
		return caps.Name
	}
	return "remote-cluster"
}

// Close cleans up router resources
func (r *Router) Close() error {
	if err := r.remoteClient.Close(); err != nil {
		return err
	}
	return nil
}

// Made with Bob
