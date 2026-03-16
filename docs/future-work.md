# Future Work: Confidence-Based Re-routing

**Status**: Planned Enhancement  
**Target**: Post-v1.0 Release  
**Dependencies**: Core cross-platform architecture and model matching system

---

## Overview

The current llm-d edge architecture implements flexible model matching (exact, substitution, family, fallback) but does not assess response quality. This document outlines planned enhancements for quality-based routing and intelligent fallback.

**Planned Features**:

1. **Runtime Confidence Scoring**: Assess each inference response for quality indicators
2. **Conditional Fallback**: Automatically retry with different models based on response confidence
3. **Advanced Model Management**: Automated model discovery, conversion, and optimization

---

## 1. Runtime Confidence Scoring

**Problem**: No mechanism to assess the quality of individual inference responses.

**Solution**: Implement per-response confidence assessment based on multiple quality indicators.

### Confidence Assessment Components

Each response will be evaluated across multiple dimensions:

1. **Completeness** (20% weight): Response not truncated, proper ending
2. **Coherence** (25% weight): Logical flow, proper grammar, readability
3. **Task Completion** (30% weight): Actually answered the question
4. **Error-Free** (20% weight): No error patterns, refusals, or hallucinations
5. **Token Probabilities** (5% weight): Model's confidence in generated tokens (if available)

### Confidence Score Output

```json
{
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
  }
}
```

### Policy-Based Thresholds

Different routing policies will have different confidence thresholds:

| Policy | Threshold | Rationale |
|--------|-----------|-----------|
| `local-first` | 0.50 | Accept moderate confidence |
| `cost-optimized` | 0.40 | Lower threshold (prioritize cost savings) |
| `latency-optimized` | 0.70 | Higher threshold (quality matters) |
| `hybrid` | 0.60 | Balanced approach |
| `remote-first` | 0.90 | Very high (rarely accept local) |
| `mobile-optimized` | 0.50 | Moderate (balance quality vs resources) |

---

## 2. Conditional Fallback Strategy

**Key Principle**: Only retry when the router substituted a model. If the user explicitly requested a specific local model, return the response as-is regardless of confidence (respect user's choice).

### Fallback Decision Flow

```
1. Execute inference with selected model
2. Assess response confidence
3. If confidence >= threshold:
   → Return response
4. If confidence < threshold:
   → Check if model was substituted
   → If NOT substituted (exact match):
      → Return response with warning (user's explicit choice)
   → If substituted:
      → Try alternate local model (if available)
      → Or fallback to remote (if enabled)
      → Return best response with metadata
```

### Example Scenarios

**Scenario 1: Successful Local with High Confidence**
- Request: `gpt-3.5-turbo` (not available locally)
- Match: `Llama-3.2-3B` can substitute
- Confidence: 0.82 (above threshold 0.60)
- Result: Return local response, no fallback needed

**Scenario 2: Low Confidence Triggers Fallback (Substituted)**
- Request: `gpt-4` (not available locally)
- Match: `Qwen3-0.6B` as fallback
- Confidence: 0.25 (below threshold 0.40)
- Model was substituted: YES
- Result: Fallback to remote with `gpt-4`

**Scenario 3: Low Confidence, No Fallback (User's Explicit Choice)**
- Request: `Llama-3.2-3B` (exact match)
- Confidence: 0.35 (below threshold 0.50)
- Model was substituted: NO
- Result: Return response with warning (user explicitly chose this model)

---

## 3. Advanced Model Manager

**Future enhancement**: Automated model lifecycle management across platforms.

### Features

1. **Model Discovery**: Automatically detect available models from HuggingFace, local storage
2. **Format Conversion**: Convert models to platform-optimal formats
   - macOS: HuggingFace → MLX
   - Windows: HuggingFace → vLLM/GGUF
   - Android/iOS: HuggingFace → GGUF/Core ML
3. **Quantization**: Automatic quantization based on device capabilities
4. **Version Control**: Track model versions and updates
5. **Storage Management**: Optimize disk usage, cleanup unused models

### Platform-Specific Converters

```go
type ModelConverter interface {
    Convert(source string, target string, format ModelFormat) error
    GetSupportedFormats() []ModelFormat
}

// Platform-specific implementations
type MLXConverter struct { /* macOS */ }
type GGUFConverter struct { /* All platforms */ }
type CoreMLConverter struct { /* iOS */ }
type ONNXConverter struct { /* Windows */ }
```

---

## 4. Enhanced API Response Metadata

Future responses will include comprehensive metadata about model selection and confidence:

```json
{
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
      "indicators": ["complete_response", "coherent_text", "task_completed", "no_errors"],
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

## Benefits

1. **Maximized Local Usage**: Use capable local models even without exact name match
2. **Quality Assurance**: Automatically detect and handle low-quality responses
3. **User Respect**: Honor explicit model choices, only retry when router substituted
4. **Transparency**: Detailed feedback about model selection and confidence
5. **Cost Optimization**: Reduce remote calls while maintaining quality standards

---

## References

For detailed specifications and implementation details, see:
- [Model Selection and Runtime Confidence Architecture](model-selection-and-confidence-architecture.md) - Complete technical specification
- [Cross-Platform Edge Device Architecture](cross-platform-llm-d-edge-architectur.md) - Current implementation

---

## Status Updates

| Date | Status | Notes |
|------|--------|-------|
| 2026-03-15 | Planned | Documented as future work after v1.0 |