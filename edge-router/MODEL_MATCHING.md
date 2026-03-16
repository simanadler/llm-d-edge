# Model Matching and Selection

This document explains the model matching and selection system implemented in the llm-d edge router.

## Overview

The edge router now supports **flexible model matching**, allowing it to use alternative local models when the exact requested model is not available. This solves the problem where developers request models like `gpt-4` or `gpt-3.5-turbo` via the OpenAI API, but only have smaller local models available.

## Key Features

1. **Flexible Matching**: Router can substitute local models based on configurable rules
2. **Match Types**: Exact, substitution, family, and fallback matching
3. **Transparent Metadata**: Response includes information about model substitution
4. **Optional Configuration**: All capabilities and matching rules are optional

## How It Works

### Matching Algorithm

When a request comes in for a model (e.g., `gpt-3.5-turbo`), the router:

1. **Exact Match** (score: 1.0): Checks if the exact model name is available locally
2. **Substitution Match** (score: 0.8): Checks if any local model has a matching substitution rule
3. **Family Match** (score: 0.7): Checks if any local model is from the same family (e.g., "llama", "qwen", "gpt"). Family is automatically extracted from model names - no configuration needed!
4. **Fallback Match** (score: 0.3): Uses any available local model as last resort

The best match (highest score, then lowest priority number) is selected.

### Configuration

#### Minimal Configuration

```yaml
models:
  local:
    - name: "meta-llama/Llama-3.2-3B"
      format: "gguf"
      quantization: "4bit"
      priority: 1
      
      matching:
        can_substitute:
          - pattern: "gpt-3.5-turbo"
          - pattern: "llama-3*"
```

#### Full Configuration

```yaml
models:
  local:
    - name: "meta-llama/Llama-3.2-3B"
      format: "gguf"
      quantization: "4bit"
      priority: 1
      
      # Optional: Model capabilities
      capabilities:
        parameter_count: "3B"
        context_length: 8192
        model_family: "llama"
        quality_tier: "medium"
        
        # Optional: Task scores
        tasks:
          chat: 0.9
          code: 0.7
          reasoning: 0.6
        
        # Optional: Domain scores
        domains:
          general: 0.9
          technical: 0.7
      
      # Optional: Matching rules
      matching:
        can_substitute:
          - pattern: "gpt-3.5*"
          - pattern: "llama*3*"
        
        exclude_patterns:
          - pattern: "gpt-4*"
```

### Pattern Matching

Patterns support wildcards (`*`):

- `gpt-3.5*` matches `gpt-3.5-turbo`, `gpt-3.5-turbo-0125`, etc.
- `*3b*` matches any model with "3b" in the name
- `llama*3*` matches `llama-3.2-3b`, `llama-3-7b`, etc.

### Response Metadata

When a model is substituted, the response includes detailed metadata:

```json
{
  "id": "chatcmpl-123",
  "model": "meta-llama/Llama-3.2-3B",
  "choices": [...],
  "usage": {...},
  "llm_d_metadata": {
    "routing_target": "local",
    "inference_engine": "mlx",
    "latency_ms": 234,
    "platform": "macos",
    "model_selection": {
      "requested_model": "gpt-3.5-turbo",
      "actual_model": "meta-llama/Llama-3.2-3B",
      "model_substituted": true,
      "match_type": "substitution",
      "match_score": 0.8
    }
  }
}
```

## Examples

### Example 1: Exact Match

**Request**: `meta-llama/Llama-3.2-3B`  
**Available**: `meta-llama/Llama-3.2-3B`  
**Result**: Exact match, score 1.0

```json
{
  "model_selection": {
    "requested_model": "meta-llama/Llama-3.2-3B",
    "actual_model": "meta-llama/Llama-3.2-3B",
    "model_substituted": false,
    "match_type": "exact",
    "match_score": 1.0
  }
}
```

### Example 2: Substitution Match

**Request**: `gpt-3.5-turbo`  
**Available**: `meta-llama/Llama-3.2-3B` (with substitution rule for `gpt-3.5*`)  
**Result**: Substitution match, score 0.8

```json
{
  "model_selection": {
    "requested_model": "gpt-3.5-turbo",
    "actual_model": "meta-llama/Llama-3.2-3B",
    "model_substituted": true,
    "match_type": "substitution",
    "match_score": 0.8
  }
}
```

### Example 3: Family Match

**Request**: `llama-3.2-7b`  
**Available**: `meta-llama/Llama-3.2-3B` (family: "llama")  
**Result**: Family match, score 0.7

```json
{
  "model_selection": {
    "requested_model": "llama-3.2-7b",
    "actual_model": "meta-llama/Llama-3.2-3B",
    "model_substituted": true,
    "match_type": "family",
    "match_score": 0.7
  }
}
```

### Example 4: Multiple Models

**Request**: `gpt-3.5-turbo`  
**Available**:
- `Qwen/Qwen3-0.6B` (priority 2, can substitute for `gpt-3.5*`)
- `meta-llama/Llama-3.2-3B` (priority 1, can substitute for `gpt-3.5*`)

**Result**: Uses `meta-llama/Llama-3.2-3B` (same score, lower priority number)

## Implementation Details

### Files Created/Modified

1. **`pkg/config/model_metadata.go`** (new)
   - Defines `ModelCapabilities`, `ModelMatching`, `SubstitutionRule`
   - Extends `LocalModelConfig` with optional capabilities and matching

2. **`pkg/router/model_matcher.go`** (new)
   - Implements `ModelMatcher` with matching algorithm
   - Supports exact, substitution, family, and fallback matching
   - Pattern matching with wildcard support

3. **`pkg/engine/types.go`** (modified)
   - Added `ModelSelectionMetadata` to response metadata
   - Includes requested model, actual model, substitution flag, match type, and score

4. **`pkg/router/router.go`** (modified)
   - Integrated `ModelMatcher` into router
   - Updated `Infer()` to populate model selection metadata
   - Modified `hasLocalModel()` to use flexible matching

5. **`pkg/config/config.go`** (modified)
   - Changed `ModelsConfig.Local` to use `ExtendedLocalModelConfig`

### Configuration Files

1. **`config.with-model-matching.yaml`** - Full example with all features
2. **`config.simple-matching.yaml`** - Minimal example with just matching rules

## Testing

To test the model matching:

1. Configure a local model with substitution rules:
```yaml
models:
  local:
    - name: "meta-llama/Llama-3.2-3B"
      matching:
        can_substitute:
          - pattern: "gpt-3.5-turbo"
```

2. Make a request for `gpt-3.5-turbo`:
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

3. Check the response metadata:
```json
{
  "llm_d_metadata": {
    "model_selection": {
      "requested_model": "gpt-3.5-turbo",
      "actual_model": "meta-llama/Llama-3.2-3B",
      "model_substituted": true,
      "match_type": "substitution",
      "match_score": 0.8
    }
  }
}
```

## Future Enhancements

The architecture document (`docs/model-selection-and-confidence-architecture.md`) describes additional features that could be implemented:

1. **Runtime Confidence Scoring**: Assess response quality for each inference
2. **Conditional Fallback**: Retry with better model if confidence is low (only when model was substituted)
3. **Adaptive Learning**: Learn from historical performance to improve matching
4. **HuggingFace Integration**: Auto-fetch model metadata from HuggingFace model cards

These features are designed but not yet implemented, allowing for future expansion.

## Made with Bob