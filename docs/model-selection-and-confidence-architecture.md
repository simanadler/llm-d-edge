# Model Selection and Runtime Confidence Architecture

**Date**: March 15, 2026  
**Author**: Planning Mode Analysis  
**Version**: 3.0 - Conditional Fallback Based on Substitution

---

## Executive Summary

This document addresses critical design flaws in the llm-d edge router's model selection and routing logic. The current implementation only routes to local models when there's an **exact name match**, and provides no mechanism to assess the quality of individual inference responses or dynamically switch models based on response confidence.

**Problems Addressed**:
1. **Rigid Model Matching**: No support for using alternative local models when exact match unavailable
2. **No Runtime Confidence Assessment**: No per-inference quality indicator to assess response confidence
3. **No Conditional Fallback**: No ability to retry with better model when router substituted a model and confidence is low

**Key Principle**: Only retry/fallback when the router substituted a different model than requested. If user explicitly requested a specific local model, return the response as-is regardless of confidence (user's choice).

**Solution Overview**:
- **Flexible Model Matching**: Support alternative local models through capability-based matching
- **Runtime Confidence Scoring**: Assess each inference response for quality/confidence indicators
- **Conditional Fallback**: Automatically retry with different model or remote **only when router substituted a model**
- **Transparent Feedback**: Return confidence scores and model substitution info to users

---

## Table of Contents

1. [Problem Analysis](#problem-analysis)
2. [Solution Architecture](#solution-architecture)
3. [Model Matching System](#model-matching-system)
4. [Runtime Confidence Scoring](#runtime-confidence-scoring)
5. [Conditional Fallback Strategy](#conditional-fallback-strategy)
6. [Model Selection Algorithm](#model-selection-algorithm)
7. [Configuration Schema](#configuration-schema)
8. [API Extensions](#api-extensions)
9. [Implementation Guidelines](#implementation-guidelines)
10. [Examples](#examples)

---

## Problem Analysis

### Current Behavior

**Scenario 1**: Developer requests `gpt-4` via OpenAI API
- **Local models available**: `Llama-3.2-3B`, `Qwen3-0.6B`
- **Current behavior**: Routes to remote (no exact match)
- **Problem**: No attempt to use capable local models

**Scenario 2**: Router substitutes model, produces low-quality response
- **Example**: User requests `gpt-4`, router uses `Qwen3-0.6B`, low confidence
- **Current behavior**: Returns low-quality response as-is
- **Problem**: No quality assessment, no retry with better model when substitution occurred

**Scenario 3**: User explicitly requests local model, produces low-quality response
- **Example**: User requests `Llama-3.2-3B` directly, low confidence
- **Desired behavior**: Return response as-is (user's explicit choice)
- **No retry**: User chose this model, respect their decision

### Root Causes

1. **Exact Match Only**: Router only uses local models with exact name match
2. **No Response Quality Assessment**: No mechanism to evaluate individual inference responses
3. **No Conditional Fallback**: Cannot retry when router substituted a model and quality is poor
4. **No Confidence Feedback**: Users don't know if response quality is degraded or if model was substituted

---

## Solution Architecture

### High-Level Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Inference Request                         │
│         (model: "gpt-4", messages: [...])                   │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│              Model Matching & Selection                      │
│  • Find candidate local models (exact, family, capability)  │
│  • Rank by suitability and availability                     │
│  • Select best candidate OR route to remote                 │
│  • Track if model was substituted                           │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│              Execute Inference (Attempt 1)                   │
│  • Run inference with selected model                        │
│  • Capture response                                         │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│         Runtime Confidence Assessment                        │
│  Analyze response for quality indicators:                   │
│  • Response completeness (not truncated)                    │
│  • Coherence (logical flow, grammar)                        │
│  • Task completion (answered the question)                  │
│  • Error patterns (refusals, errors, hallucinations)        │
│  • Token probabilities (if available)                       │
│  → Confidence Score: 0.0 - 1.0                              │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ├─ Confidence >= Threshold ─────────────┐
                     │                                        │
                     │                                        ▼
                     │                          ┌─────────────────────┐
                     │                          │  Return Response    │
                     │                          │  + Confidence Score │
                     │                          └─────────────────────┘
                     │
                     └─ Confidence < Threshold
                                │
┌───────────────────────────────▼─────────────────────────────┐
│         Check if Model Was Substituted                       │
│  • Was requested model != actual model used?                │
│  • If NO substitution: Return response (user's choice)      │
│  • If YES substitution: Proceed to fallback                 │
└────────────────────┬────────────────────────────────────────┘
                     │
                     └─ Model Was Substituted
                                │
┌───────────────────────────────▼─────────────────────────────┐
│              Dynamic Fallback Decision                       │
│  Options based on policy:                                   │
│  1. Try next best local model (if available)                │
│  2. Re-route to remote (original requested model)           │
│  3. Return low-confidence response with warning             │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│              Execute Fallback (Attempt 2)                    │
│  • Run inference with fallback model/remote                 │
│  • Assess confidence again                                  │
│  • Return best response (or both with confidence scores)    │
└─────────────────────────────────────────────────────────────┘
```

### Design Principles

1. **Runtime Assessment**: Evaluate each response individually, not just model capabilities
2. **Conditional Fallback**: Only retry when router substituted a model (respect user's explicit choices)
3. **Policy-Driven Thresholds**: Confidence thresholds vary by routing policy
4. **Transparent Feedback**: Always inform user about model switches and confidence
5. **Performance Conscious**: Minimize latency overhead of confidence assessment

---

## Model Matching System

### Model Metadata Schema

Extend model configuration with capability metadata:

```yaml
models:
  local:
    - name: "meta-llama/Llama-3.2-3B"
      format: "gguf"
      quantization: "4bit"
      priority: 1
      
      # Model capabilities for matching
      capabilities:
        parameter_count: "3B"
        context_length: 8192
        model_family: "llama"
        quality_tier: "medium"  # low, medium, high, premium
        
        # Task suitability scores (0.0-1.0)
        tasks:
          chat: 0.9
          code: 0.7
          reasoning: 0.6
          creative_writing: 0.8
          summarization: 0.85
      
      # Substitution rules
      matching:
        can_substitute:
          - pattern: "gpt-3.5.*"
          - pattern: "llama.*3.*"
        
        exclude_patterns:
          - "gpt-4.*"
          - "claude-3-opus.*"
```

### Matching Algorithm

```go
type ModelMatcher struct {
    localModels []ModelMetadata
    logger      *zap.Logger
}

// FindCandidates returns ranked list of local models that could handle request
func (m *ModelMatcher) FindCandidates(requestedModel string) []ModelCandidate {
    candidates := []ModelCandidate{}
    
    // 1. Exact match (highest priority)
    for _, model := range m.localModels {
        if model.Name == requestedModel {
            candidates = append(candidates, ModelCandidate{
                Model:      model,
                MatchType:  "exact",
                MatchScore: 1.0,
            })
            return candidates  // Return immediately for exact match
        }
    }
    
    // 2. Substitution match (check matching rules)
    for _, model := range m.localModels {
        for _, rule := range model.Matching.CanSubstitute {
            if matchesPattern(requestedModel, rule.Pattern) {
                candidates = append(candidates, ModelCandidate{
                    Model:      model,
                    MatchType:  "substitution",
                    MatchScore: rule.MinConfidence,
                })
            }
        }
    }
    
    // 3. Family match (e.g., llama-3.2-3b for llama-3.2-7b)
    requestedFamily := extractModelFamily(requestedModel)
    for _, model := range m.localModels {
        if model.Capabilities.ModelFamily == requestedFamily {
            candidates = append(candidates, ModelCandidate{
                Model:      model,
                MatchType:  "family",
                MatchScore: 0.7,
            })
        }
    }
    
    // 4. Fallback: any available model (lowest priority)
    if len(candidates) == 0 {
        for _, model := range m.localModels {
            candidates = append(candidates, ModelCandidate{
                Model:      model,
                MatchType:  "fallback",
                MatchScore: 0.3,
            })
        }
    }
    
    // Sort by match score (descending) and priority
    sort.Slice(candidates, func(i, j int) bool {
        if candidates[i].MatchScore != candidates[j].MatchScore {
            return candidates[i].MatchScore > candidates[j].MatchScore
        }
        return candidates[i].Model.Priority < candidates[j].Model.Priority
    })
    
    return candidates
}
```

---

## Runtime Confidence Scoring

### Confidence Assessment Engine

The key innovation: **assess each inference response** for quality indicators.

```go
type ConfidenceAssessor struct {
    logger *zap.Logger
}

type ConfidenceScore struct {
    Overall     float64  // 0.0-1.0: Overall confidence in response
    Components  ConfidenceComponents
    Indicators  []string // Human-readable quality indicators
    Issues      []string // Detected issues
}

type ConfidenceComponents struct {
    Completeness    float64  // Response not truncated
    Coherence       float64  // Logical flow, grammar
    TaskCompletion  float64  // Answered the question
    ErrorFree       float64  // No error patterns
    TokenProbs      float64  // Token probability confidence (if available)
}

// AssessResponse evaluates the quality/confidence of an inference response
func (c *ConfidenceAssessor) AssessResponse(
    response *InferenceResponse,
    request *InferenceRequest,
) *ConfidenceScore {
    
    components := ConfidenceComponents{}
    indicators := []string{}
    issues := []string{}
    
    responseText := response.Choices[0].Message.Content
    
    // 1. Completeness check
    components.Completeness = c.assessCompleteness(response, request)
    if components.Completeness < 1.0 {
        issues = append(issues, "response_truncated")
    } else {
        indicators = append(indicators, "complete_response")
    }
    
    // 2. Coherence check
    components.Coherence = c.assessCoherence(responseText)
    if components.Coherence < 0.7 {
        issues = append(issues, "low_coherence")
    } else {
        indicators = append(indicators, "coherent_text")
    }
    
    // 3. Task completion check
    components.TaskCompletion = c.assessTaskCompletion(responseText, request)
    if components.TaskCompletion < 0.6 {
        issues = append(issues, "task_incomplete")
    } else {
        indicators = append(indicators, "task_completed")
    }
    
    // 4. Error pattern detection
    components.ErrorFree = c.checkErrorPatterns(responseText)
    if components.ErrorFree < 1.0 {
        issues = append(issues, "error_patterns_detected")
    } else {
        indicators = append(indicators, "no_errors")
    }
    
    // 5. Token probabilities (if available from model)
    components.TokenProbs = c.assessTokenProbabilities(response)
    if components.TokenProbs > 0 {
        if components.TokenProbs < 0.5 {
            issues = append(issues, "low_token_confidence")
        } else {
            indicators = append(indicators, "high_token_confidence")
        }
    }
    
    // Calculate overall confidence (weighted average)
    overall := c.calculateOverallConfidence(components)
    
    return &ConfidenceScore{
        Overall:    overall,
        Components: components,
        Indicators: indicators,
        Issues:     issues,
    }
}

// calculateOverallConfidence combines component scores
func (c *ConfidenceAssessor) calculateOverallConfidence(comp ConfidenceComponents) float64 {
    // Weighted average of components
    weights := map[string]float64{
        "completeness":   0.20,
        "coherence":      0.25,
        "taskCompletion": 0.30,
        "errorFree":      0.20,
        "tokenProbs":     0.05,  // Lower weight (not always available)
    }
    
    score := (comp.Completeness * weights["completeness"] +
              comp.Coherence * weights["coherence"] +
              comp.TaskCompletion * weights["taskCompletion"] +
              comp.ErrorFree * weights["errorFree"] +
              comp.TokenProbs * weights["tokenProbs"])
    
    return score
}
```

### Confidence Thresholds by Policy

```go
// GetConfidenceThreshold returns minimum acceptable confidence for a policy
func GetConfidenceThreshold(policy RoutingPolicy) float64 {
    switch policy {
    case PolicyLocalFirst:
        return 0.50  // Accept moderate confidence
    
    case PolicyCostOptimized:
        return 0.40  // Lower threshold (prioritize cost savings)
    
    case PolicyLatencyOptimized:
        return 0.70  // Higher threshold (quality matters)
    
    case PolicyHybrid:
        return 0.60  // Balanced
    
    case PolicyRemoteFirst:
        return 0.90  // Very high (rarely accept local)
    
    case PolicyMobileOptimized:
        return 0.50  // Moderate (balance quality vs resources)
    
    default:
        return 0.60
    }
}
```

---

## Conditional Fallback Strategy

### Fallback Decision Engine

**CRITICAL**: Only retry when router substituted a model. Respect user's explicit model choices.

```go
type FallbackManager struct {
    matcher    *ModelMatcher
    assessor   *ConfidenceAssessor
    remote     *RemoteClient
    policy     RoutingPolicy
    logger     *zap.Logger
}

type FallbackConfig struct {
    Enabled              bool
    MaxAttempts          int      // Max retry attempts
    ConfidenceThreshold  float64  // Min acceptable confidence
    TryAlternateLocal    bool     // Try other local models first
    FallbackToRemote     bool     // Fallback to remote if needed
    ReturnAllAttempts    bool     // Return all attempts with scores
}

type InferenceAttempt struct {
    Model      string
    Response   *InferenceResponse
    Confidence *ConfidenceScore
    Target     RouteTarget  // local or remote
    AttemptNum int
}

// ExecuteWithFallback performs inference with automatic fallback
// IMPORTANT: Only retries if model was substituted by router
func (f *FallbackManager) ExecuteWithFallback(
    ctx context.Context,
    req *InferenceRequest,
    config FallbackConfig,
) (*InferenceResponse, error) {
    
    attempts := []InferenceAttempt{}
    threshold := config.ConfidenceThreshold
    requestedModel := req.Model
    
    // Get candidate models
    candidates := f.matcher.FindCandidates(requestedModel)
    
    // Determine if we have an exact match
    hasExactMatch := len(candidates) > 0 && candidates[0].MatchType == "exact"
    
    // Attempt 1: Try best local candidate
    if len(candidates) > 0 {
        attempt1 := f.tryLocalModel(ctx, req, candidates[0].Model)
        attempts = append(attempts, attempt1)
        
        modelWasSubstituted := (candidates[0].Model.Name != requestedModel)
        
        // Check confidence
        if attempt1.Confidence.Overall >= threshold {
            // Success! Return response
            return f.finalizeResponse(attempt1, attempts, config, modelWasSubstituted), nil
        }
        
        // Low confidence detected
        f.logger.Warn("local inference confidence below threshold",
            zap.String("requested_model", requestedModel),
            zap.String("actual_model", attempt1.Model),
            zap.Bool("substituted", modelWasSubstituted),
            zap.Float64("confidence", attempt1.Confidence.Overall),
            zap.Float64("threshold", threshold),
            zap.Strings("issues", attempt1.Confidence.Issues))
        
        // CRITICAL: Only retry if model was substituted
        if !modelWasSubstituted {
            // User explicitly requested this model - return as-is with warning
            f.logger.Info("returning low-confidence response (user's explicit model choice)",
                zap.String("model", attempt1.Model),
                zap.Float64("confidence", attempt1.Confidence.Overall))
            
            return f.finalizeResponse(attempt1, attempts, config, false), nil
        }
        
        // Model was substituted and confidence low - try fallback
    }
    
    // Attempt 2: Try alternate local model (if enabled, available, and substitution occurred)
    if config.TryAlternateLocal && len(candidates) > 1 {
        attempt2 := f.tryLocalModel(ctx, req, candidates[1].Model)
        attempts = append(attempts, attempt2)
        
        if attempt2.Confidence.Overall >= threshold {
            // Alternate model succeeded
            return f.finalizeResponse(attempt2, attempts, config, true), nil
        }
        
        f.logger.Warn("alternate local model confidence below threshold",
            zap.String("model", attempt2.Model),
            zap.Float64("confidence", attempt2.Confidence.Overall))
    }
    
    // Attempt 3: Fallback to remote (if enabled and substitution occurred)
    if config.FallbackToRemote {
        attempt3 := f.tryRemote(ctx, req)
        attempts = append(attempts, attempt3)
        
        // Remote is assumed high confidence (no assessment needed)
        return f.finalizeResponse(attempt3, attempts, config, true), nil
    }
    
    // No fallback available - return best attempt with warning
    bestAttempt := f.selectBestAttempt(attempts)
    modelWasSubstituted := (bestAttempt.Model != requestedModel)
    return f.finalizeResponse(bestAttempt, attempts, config, modelWasSubstituted), nil
}

// tryLocalModel executes inference with a local model and assesses confidence
func (f *FallbackManager) tryLocalModel(
    ctx context.Context,
    req *InferenceRequest,
    model ModelMetadata,
) InferenceAttempt {
    
    // Execute inference
    response, err := f.executeLocal(ctx, req, model)
    if err != nil {
        return InferenceAttempt{
            Model:  model.Name,
            Target: TargetLocal,
            Confidence: &ConfidenceScore{
                Overall: 0.0,
                Issues:  []string{"execution_failed: " + err.Error()},
            },
        }
    }
    
    // Assess confidence
    confidence := f.assessor.AssessResponse(response, req)
    
    return InferenceAttempt{
        Model:      model.Name,
        Response:   response,
        Confidence: confidence,
        Target:     TargetLocal,
    }
}

// tryRemote executes inference with remote cluster
func (f *FallbackManager) tryRemote(
    ctx context.Context,
    req *InferenceRequest,
) InferenceAttempt {
    
    response, err := f.remote.Infer(ctx, req)
    if err != nil {
        return InferenceAttempt{
            Model:  req.Model,
            Target: TargetRemote,
            Confidence: &ConfidenceScore{
                Overall: 0.0,
                Issues:  []string{"remote_failed: " + err.Error()},
            },
        }
    }
    
    // Remote responses assumed high confidence
    return InferenceAttempt{
        Model:    req.Model,
        Response: response,
        Confidence: &ConfidenceScore{
            Overall:    1.0,
            Indicators: []string{"remote_inference"},
        },
        Target: TargetRemote,
    }
}

// finalizeResponse prepares final response with metadata
func (f *FallbackManager) finalizeResponse(
    selected InferenceAttempt,
    allAttempts []InferenceAttempt,
    config FallbackConfig,
    modelWasSubstituted bool,
) *InferenceResponse {
    
    response := selected.Response
    requestedModel := allAttempts[0].Model
    
    // Add comprehensive metadata
    response.Metadata.RequestedModel = requestedModel
    response.Metadata.ActualModel = selected.Model
    response.Metadata.ModelSubstituted = modelWasSubstituted
    response.Metadata.RoutingTarget = string(selected.Target)
    
    // Add confidence information
    response.Metadata.Confidence = ConfidenceMetadata{
        Score:      selected.Confidence.Overall,
        Components: selected.Confidence.Components,
        Indicators: selected.Confidence.Indicators,
        Issues:     selected.Confidence.Issues,
    }
    
    // Add attempt history
    if config.ReturnAllAttempts {
        response.Metadata.Attempts = f.summarizeAttempts(allAttempts)
    }
    
    // Add warnings based on confidence and substitution
    if selected.Confidence.Overall < config.ConfidenceThreshold {
        if modelWasSubstituted {
            response.Metadata.Warnings = []string{
                fmt.Sprintf("Response confidence (%.2f) below threshold (%.2f). Router substituted %s for requested %s.",
                    selected.Confidence.Overall, config.ConfidenceThreshold, selected.Model, requestedModel),
            }
        } else {
            response.Metadata.Warnings = []string{
                fmt.Sprintf("Response confidence (%.2f) below threshold (%.2f). This was your explicitly requested model.",
                    selected.Confidence.Overall, config.ConfidenceThreshold),
            }
        }
    }
    
    return response
}
```

---

## Model Selection Algorithm

### Integrated Selection with Confidence

```go
type Router struct {
    matcher  *ModelMatcher
    assessor *ConfidenceAssessor
    fallback *FallbackManager
    policy   RoutingPolicy
    config   *Config
    logger   *zap.Logger
}

// Route determines routing and executes inference with confidence assessment
func (r *Router) Route(
    ctx context.Context,
    req *InferenceRequest,
) (*InferenceResponse, error) {
    
    // 1. Find candidate local models
    candidates := r.matcher.FindCandidates(req.Model)
    
    // 2. Determine if we should try local first
    shouldTryLocal := r.shouldTryLocal(candidates, req)
    
    if !shouldTryLocal {
        // Route directly to remote
        return r.fallback.tryRemote(ctx, req).Response, nil
    }
    
    // 3. Execute with fallback (includes confidence assessment)
    fallbackConfig := FallbackConfig{
        Enabled:             r.config.Edge.ModelSelection.Fallback.Enabled,
        MaxAttempts:         r.config.Edge.ModelSelection.Fallback.MaxRetries + 1,
        ConfidenceThreshold: r.getConfidenceThreshold(),
        TryAlternateLocal:   r.config.Edge.ModelSelection.Fallback.TryAlternateLocal,
        FallbackToRemote:    r.config.Edge.ModelSelection.Fallback.FallbackToRemote,
        ReturnAllAttempts:   false,
    }
    
    return r.fallback.ExecuteWithFallback(ctx, req, fallbackConfig)
}

// shouldTryLocal determines if we should attempt local inference
func (r *Router) shouldTryLocal(candidates []ModelCandidate, req *InferenceRequest) bool {
    if len(candidates) == 0 {
        return false  // No local models available
    }
    
    // Check network connectivity
    if !r.isConnected() {
        return true  // Must use local (offline)
    }
    
    // Apply policy
    switch r.policy {
    case PolicyLocalFirst, PolicyCostOptimized:
        return true  // Always try local first
    
    case PolicyRemoteFirst:
        return false  // Skip local
    
    case PolicyHybrid:
        // Try local for small requests
        return r.estimatePromptTokens(req) < 1000
    
    case PolicyLatencyOptimized:
        // Try local for interactive requests
        return req.Stream
    
    case PolicyMobileOptimized:
        // Try local if battery/thermal OK
        return !r.isBatteryLow() && !r.isThermalThrottling()
    
    default:
        return true
    }
}

// getConfidenceThreshold returns threshold for current policy
func (r *Router) getConfidenceThreshold() float64 {
    // Check for custom threshold in config
    if threshold, ok := r.config.Edge.ModelSelection.ConfidenceThresholds[r.policy]; ok {
        return threshold
    }
    
    // Use default for policy
    return GetConfidenceThreshold(r.policy)
}
```

---

## Configuration Schema

### Enhanced Configuration

```yaml
edge:
  routing:
    policy: "hybrid"  # local-first, remote-first, hybrid, cost-optimized, etc.
    fallback: "remote"
  
  # NEW: Model selection and confidence configuration
  model_selection:
    # Enable flexible model matching
    enable_substitution: true
    
    # Confidence thresholds by policy (override defaults)
    confidence_thresholds:
      local-first: 0.50
      cost-optimized: 0.40
      latency-optimized: 0.70
      hybrid: 0.60
      remote-first: 0.90
      mobile-optimized: 0.50
    
    # Fallback configuration
    fallback:
      enabled: true
      max_retries: 2
      try_alternate_local: true
      fallback_to_remote: true
      return_all_attempts: false  # Include all attempts in response metadata
    
    # Confidence assessment configuration
    confidence_assessment:
      enabled: true
      
      # Component weights (must sum to 1.0)
      weights:
        completeness: 0.20
        coherence: 0.25
        task_completion: 0.30
        error_free: 0.20
        token_probs: 0.05
      
      # Detection thresholds
      thresholds:
        min_response_length: 10
        max_repetition_ratio: 0.3
        min_coherence_score: 0.6
  
  models:
    local:
      - name: "meta-llama/Llama-3.2-3B"
        format: "gguf"
        quantization: "4bit"
        priority: 1
        
        capabilities:
          parameter_count: "3B"
          context_length: 8192
          model_family: "llama"
          quality_tier: "medium"
          
          tasks:
            chat: 0.9
            code: 0.7
            reasoning: 0.6
        
        matching:
          can_substitute:
            - pattern: "gpt-3.5.*"
              min_confidence: 0.6
            - pattern: "llama.*3.*"
              min_confidence: 0.8
          
          exclude_patterns:
            - "gpt-4.*"
      
      - name: "Qwen/Qwen3-0.6B"
        format: "gguf"
        quantization: "4bit"
        priority: 2
        
        capabilities:
          parameter_count: "0.6B"
          context_length: 4096
          model_family: "qwen"
          quality_tier: "low"
          
          tasks:
            chat: 0.7
            code: 0.5
        
        matching:
          can_substitute:
            - pattern: "gpt-3.5-turbo-0125"
              min_confidence: 0.5
          
          exclude_patterns:
            - "gpt-4.*"
            - ".*7b.*"
    
    remote:
      cluster_url: "https://llm-d.example.com"
      auth_token: "${LLMD_AUTH_TOKEN}"
```

---

## API Extensions

### Enhanced Response Metadata

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "Llama-3.2-3B",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help you today?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 5,
    "completion_tokens": 8,
    "total_tokens": 13
  },
  "llm_d_metadata": {
    "routing_target": "local",
    "inference_engine": "mlx",
    "latency_ms": 234,
    
    "model_selection": {
      "requested_model": "gpt-4",
      "actual_model": "meta-llama/Llama-3.2-3B",
      "model_substituted": true,
      "match_type": "substitution"
    },
    
    "confidence": {
      "score": 0.82,
      "components": {
        "completeness": 1.0,
        "coherence": 0.85,
        "task_completion": 0.80,
        "error_free": 1.0,
        "token_probs": 0.0
      },
      "indicators": [
        "complete_response",
        "coherent_text",
        "task_completed",
        "no_errors"
      ],
      "issues": []
    },
    
    "attempts": [
      {
        "model": "meta-llama/Llama-3.2-3B",
        "target": "local",
        "confidence": 0.82,
        "selected": true
      }
    ],
    
    "warnings": []
  }
}
```

---

## Implementation Guidelines

### Phase 1: Model Matching (Week 1-2)

**Files to Create/Modify**:
- [`edge-router/pkg/router/model_matcher.go`](edge-router/pkg/router/model_matcher.go:1) - New file
- [`edge-router/pkg/config/config.go`](edge-router/pkg/config/config.go:1) - Add model capabilities
- [`edge-router/config.example.yaml`](edge-router/config.example.yaml:1) - Add matching rules

**Tasks**:
1. Define `ModelMetadata` struct with capabilities
2. Implement `ModelMatcher` with matching algorithms
3. Add pattern matching for substitution rules
4. Update configuration schema

### Phase 2: Confidence Assessment (Week 3-4)

**Files to Create/Modify**:
- [`edge-router/pkg/router/confidence_assessor.go`](edge-router/pkg/router/confidence_assessor.go:1) - New file
- [`edge-router/pkg/engine/types.go`](edge-router/pkg/engine/types.go:110) - Extend response metadata

**Tasks**:
1. Implement `ConfidenceAssessor` with component scoring
2. Add completeness, coherence, task completion checks
3. Implement error pattern detection
4. Add token probability assessment (if available)

### Phase 3: Conditional Fallback (Week 5-6)

**Files to Create/Modify**:
- [`edge-router/pkg/router/fallback_manager.go`](edge-router/pkg/router/fallback_manager.go:1) - New file
- [`edge-router/pkg/router/router.go`](edge-router/pkg/router/router.go:58) - Integrate fallback

**Tasks**:
1. Implement `FallbackManager` with conditional retry logic
2. Add model substitution tracking
3. Implement alternate local model attempts (only when substituted)
4. Implement remote fallback (only when substituted)
5. Add attempt tracking and metadata

### Phase 4: Integration & Testing (Week 7-8)

**Files to Modify**:
- [`edge-router/pkg/router/router.go`](edge-router/pkg/router/router.go:58) - Main integration
- [`edge-router/cmd/edge-router/main.go`](edge-router/cmd/edge-router/main.go:1) - Wire components

**Tasks**:
1. Integrate all components into router
2. Add comprehensive unit tests
3. Add integration tests with mock models
4. Test conditional fallback logic (substituted vs explicit)
5. Performance testing and optimization

---

## Examples

### Example 1: Successful Local with High Confidence

**Request**: `gpt-3.5-turbo` (not available locally)  
**Local Models**: `Llama-3.2-3B` (can substitute)  
**Policy**: `hybrid` (threshold: 0.60)

**Flow**:
1. Match: `Llama-3.2-3B` can substitute for `gpt-3.5-turbo` (rule confidence: 0.70)
2. Execute: Local inference with `Llama-3.2-3B`
3. Assess: Confidence score 0.82 (complete, coherent, task completed)
4. Decision: Return response (0.82 >= 0.60)

**Result**: Local response with high confidence, no fallback needed

### Example 2: Low Confidence Triggers Fallback (Model Substituted)

**Request**: `gpt-4` (not available locally)  
**Local Models**: `Qwen3-0.6B` (fallback match)  
**Policy**: `cost-optimized` (threshold: 0.40)

**Flow**:
1. Match: `Qwen3-0.6B` as fallback (match score: 0.30)
2. Execute: Local inference with `Qwen3-0.6B`
3. Assess: Confidence score 0.25 (low coherence, incomplete)
4. Check: Model was substituted (`Qwen3-0.6B` != `gpt-4`)
5. Decision: Below threshold (0.25 < 0.40) AND substituted → Fallback
6. Fallback: Try remote with `gpt-4`
7. Return: Remote response with fallback metadata

**Result**: Remote response after substituted local model failed confidence check

### Example 3: Alternate Local Model Success

**Request**: `gpt-3.5-turbo`  
**Local Models**: `Qwen3-0.6B` (priority 1), `Llama-3.2-3B` (priority 2)  
**Policy**: `local-first` (threshold: 0.50)

**Flow**:
1. Match: Both models can substitute
2. Attempt 1: `Qwen3-0.6B` → Confidence 0.35 (below threshold)
3. Check: Model was substituted → Try alternate
4. Attempt 2: `Llama-3.2-3B` → Confidence 0.75 (above threshold)
5. Return: `Llama-3.2-3B` response

**Result**: Second local model succeeded, avoided remote call

### Example 4: Low Confidence, No Fallback (User's Explicit Choice)

**Request**: `Llama-3.2-3B` (available locally, exact match)  
**Local Models**: `Llama-3.2-3B`, `Qwen3-0.6B`  
**Policy**: `local-first` (threshold: 0.50)

**Flow**:
1. Match: Exact match for `Llama-3.2-3B`
2. Execute: Local inference with `Llama-3.2-3B`
3. Assess: Confidence score 0.35 (below threshold)
4. Check: Model was NOT substituted (exact match)
5. Decision: Return response as-is (user's explicit choice)
6. Return: Response with low confidence warning

**Response Metadata**:
```json
{
  "model_selection": {
    "requested_model": "Llama-3.2-3B",
    "actual_model": "Llama-3.2-3B",
    "model_substituted": false
  },
  "confidence": {
    "score": 0.35,
    "issues": ["low_coherence", "task_incomplete"]
  },
  "warnings": [
    "Response confidence (0.35) below threshold (0.50). This was your explicitly requested model."
  ]
}
```

**Result**: Low-confidence response returned without retry (user explicitly chose this model)

---

## Summary

This architecture solves all three problems:

1. **✅ Flexible Model Matching**: Supports exact, substitution, family, and fallback matching
2. **✅ Runtime Confidence Scoring**: Assesses each inference response for quality indicators
3. **✅ Conditional Fallback**: Retries only when router substituted a model (respects user's explicit choices)

**Key Innovations**:
- **Per-Inference Assessment**: Every response evaluated individually
- **Conditional Retry**: Only retry when router substituted a model
- **Respect User Choice**: Never retry when user explicitly requested a specific model
- **Transparent Feedback**: Detailed confidence scores and attempt history
- **Policy-Driven**: Thresholds and behavior adapt to routing policy

**Benefits**:
- Maximizes local inference usage while maintaining quality
- Provides clear feedback about model substitutions and confidence
- Enables intelligent fallback based on actual response quality
- Respects user's explicit model choices
- Adapts behavior based on routing policies and user preferences

**Next Steps**:
1. Review and approve architecture
2. Begin Phase 1 implementation (model matching)
3. Iterate based on real-world testing