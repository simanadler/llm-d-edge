# Model Metadata Sources and Acquisition Strategy

**Date**: March 15, 2026  
**Author**: Planning Mode Analysis  
**Version**: 1.0

---

## Overview

This document defines how model capability metadata is sourced and managed for the llm-d edge router's intelligent model selection system.

**Strategy**: HuggingFace Model Cards + Manual Override
- **Auto-fetch**: Basic model information from HuggingFace
- **Manual config**: Task-specific capabilities and substitution rules
- **Adaptive learning**: Optional runtime performance tracking

---

## Metadata Categories

### 1. Auto-Fetchable from HuggingFace

These fields can be automatically extracted from HuggingFace model cards:

```yaml
capabilities:
  # From model card metadata
  parameter_count: "3B"           # model.safetensors.metadata or config.json
  context_length: 8192            # config.json: max_position_embeddings
  model_family: "llama"           # Inferred from model name/architecture
  architecture: "LlamaForCausalLM" # config.json: architectures
  
  # From model card tags
  languages: ["en", "es", "fr"]   # model card: language tags
  license: "apache-2.0"           # model card: license
  
  # From model card metadata
  base_model: "meta-llama/Llama-3.2-3B-Instruct"  # If fine-tuned
```

**HuggingFace API Example**:
```python
from huggingface_hub import model_info

info = model_info("meta-llama/Llama-3.2-3B")
# info.config contains model configuration
# info.tags contains model tags
# info.card_data contains model card metadata
```

### 2. Manual Configuration Required

These fields require human judgment or testing:

```yaml
capabilities:
  # Quality assessment (requires testing/benchmarks)
  quality_tier: "medium"  # low, medium, high, premium
  
  # Task suitability scores (0.0-1.0)
  tasks:
    chat: 0.9              # Conversational ability
    code: 0.7              # Code generation/understanding
    reasoning: 0.6         # Logical reasoning
    creative_writing: 0.8  # Creative text generation
    summarization: 0.85    # Text summarization
    translation: 0.7       # Language translation
    math: 0.5              # Mathematical reasoning
    analysis: 0.7          # Data/text analysis
  
  # Domain expertise (0.0-1.0)
  domains:
    general: 0.9           # General knowledge
    technical: 0.7         # Technical/programming
    medical: 0.3           # Medical knowledge
    legal: 0.3             # Legal knowledge
    scientific: 0.6        # Scientific knowledge

# Substitution rules (requires understanding of model capabilities)
matching:
  can_substitute:
    - pattern: "gpt-3.5.*"
    - pattern: "llama.*3.*"
  
  exclude_patterns:
    - "gpt-4.*"
    - "claude-3-opus.*"
```

### 3. Optional: Benchmark-Based Initialization

Use public benchmark results to initialize task scores:

**Common Benchmarks**:
- **MMLU** (Massive Multitask Language Understanding) → `reasoning`, `general`
- **HumanEval** → `code`
- **GSM8K** → `math`
- **HellaSwag** → `reasoning`
- **TruthfulQA** → `general`, `analysis`

**Example Mapping**:
```yaml
# If model has MMLU score of 65%
tasks:
  reasoning: 0.65
  general: 0.65

# If model has HumanEval score of 45%
tasks:
  code: 0.45
```

---

## Implementation Strategy

### Phase 1: Manual Configuration (MVP)

**Approach**: Users manually configure all metadata

**Pros**:
- Simple to implement
- No external dependencies
- Full control over values

**Cons**:
- Time-consuming for users
- Requires knowledge of model capabilities
- No standardization

**Example**:
```yaml
models:
  local:
    - name: "meta-llama/Llama-3.2-3B"
      # User manually fills everything
      capabilities:
        parameter_count: "3B"
        context_length: 8192
        model_family: "llama"
        quality_tier: "medium"
        tasks:
          chat: 0.9
          code: 0.7
```

### Phase 2: HuggingFace Auto-Fetch (Recommended)

**Approach**: Auto-fetch basic info, manual for task scores

**Implementation**:
```go
type ModelMetadataFetcher struct {
    hfClient *huggingface.Client
    cache    *MetadataCache
}

// FetchModelMetadata retrieves metadata from HuggingFace
func (f *ModelMetadataFetcher) FetchModelMetadata(modelName string) (*ModelMetadata, error) {
    // Check cache first
    if cached := f.cache.Get(modelName); cached != nil {
        return cached, nil
    }
    
    // Fetch from HuggingFace
    info, err := f.hfClient.ModelInfo(modelName)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch model info: %w", err)
    }
    
    metadata := &ModelMetadata{
        Name: modelName,
        Capabilities: ModelCapabilities{
            // Auto-populated from HF
            ParameterCount:  extractParameterCount(info),
            ContextLength:   extractContextLength(info),
            ModelFamily:     inferModelFamily(modelName, info),
            Architecture:    info.Config.Architectures[0],
            Languages:       info.Tags.Languages,
            
            // Defaults (to be overridden manually)
            QualityTier: inferQualityTier(extractParameterCount(info)),
            Tasks:       getDefaultTaskScores(extractParameterCount(info)),
            Domains:     getDefaultDomainScores(),
        },
    }
    
    // Cache for future use
    f.cache.Set(modelName, metadata)
    
    return metadata, nil
}

// extractParameterCount parses parameter count from model info
func extractParameterCount(info *ModelInfo) string {
    // Try config.json first
    if params := info.Config.NumParameters; params > 0 {
        return formatParameterCount(params)
    }
    
    // Try model card
    if params := info.CardData.ModelSize; params != "" {
        return params
    }
    
    // Infer from model name
    return inferParameterCount(info.ModelID)
}

// getDefaultTaskScores provides conservative defaults based on model size
func getDefaultTaskScores(paramCount string) map[string]float64 {
    size := parseParameterCount(paramCount)
    
    if size < 1.0 {  // < 1B
        return map[string]float64{
            "chat": 0.6, "code": 0.4, "reasoning": 0.3,
            "creative_writing": 0.5, "summarization": 0.6,
        }
    } else if size < 7.0 {  // 1B-7B
        return map[string]float64{
            "chat": 0.8, "code": 0.6, "reasoning": 0.5,
            "creative_writing": 0.7, "summarization": 0.8,
        }
    } else {  // 7B+
        return map[string]float64{
            "chat": 0.9, "code": 0.7, "reasoning": 0.7,
            "creative_writing": 0.8, "summarization": 0.9,
        }
    }
}
```

**Configuration with Auto-Fetch**:
```yaml
models:
  local:
    - name: "meta-llama/Llama-3.2-3B"
      # Auto-fetched from HuggingFace (optional overrides)
      capabilities:
        # These are auto-populated, but can be overridden
        # parameter_count: "3B"  # Auto-fetched
        # context_length: 8192   # Auto-fetched
        # model_family: "llama"  # Auto-fetched
        
        # Manual configuration (required)
        quality_tier: "medium"
        tasks:
          chat: 0.9
          code: 0.7
          reasoning: 0.6
```

### Phase 3: Adaptive Learning (Future)

**Approach**: Learn from runtime performance

**Implementation**:
```go
type AdaptiveLearner struct {
    history *PerformanceHistory
    logger  *zap.Logger
}

// UpdateTaskScores adjusts task scores based on actual performance
func (a *AdaptiveLearner) UpdateTaskScores(
    modelName string,
    taskType string,
    confidenceScore float64,
) {
    // Get current score
    currentScore := a.getTaskScore(modelName, taskType)
    
    // Get historical average
    historicalAvg := a.history.GetAverageConfidence(modelName, taskType)
    
    // Blend: 80% historical, 20% current
    newScore := 0.8*historicalAvg + 0.2*confidenceScore
    
    // Update score (with bounds)
    newScore = math.Max(0.0, math.Min(1.0, newScore))
    
    a.logger.Info("updated task score",
        zap.String("model", modelName),
        zap.String("task", taskType),
        zap.Float64("old_score", currentScore),
        zap.Float64("new_score", newScore))
    
    // Persist updated score
    a.persistTaskScore(modelName, taskType, newScore)
}
```

---

## Configuration Examples

### Example 1: Minimal (Auto-Fetch Only)

```yaml
models:
  local:
    - name: "meta-llama/Llama-3.2-3B"
      format: "gguf"
      quantization: "4bit"
      # Everything else auto-fetched from HuggingFace
```

### Example 2: Hybrid (Auto-Fetch + Manual Overrides)

```yaml
models:
  local:
    - name: "meta-llama/Llama-3.2-3B"
      format: "gguf"
      quantization: "4bit"
      
      # Override auto-fetched values
      capabilities:
        quality_tier: "high"  # Override default "medium"
        
        # Fine-tune task scores based on testing
        tasks:
          chat: 0.95           # Better than default
          code: 0.75           # Better than default
          reasoning: 0.65      # Better than default
      
      # Custom substitution rules
      matching:
        can_substitute:
          - pattern: "gpt-3.5.*"
            min_confidence: 0.7  # Higher confidence required
```

### Example 3: Full Manual Control

```yaml
models:
  local:
    - name: "custom-model-v1"
      format: "gguf"
      quantization: "4bit"
      
      # Fully manual (no auto-fetch)
      capabilities:
        parameter_count: "2.7B"
        context_length: 4096
        model_family: "custom"
        quality_tier: "medium"
        
        tasks:
          chat: 0.85
          code: 0.60
          reasoning: 0.55
          creative_writing: 0.75
        
        domains:
          general: 0.80
          technical: 0.65
      
      matching:
        can_substitute:
          - pattern: "gpt-3.5-turbo-0125"
            min_confidence: 0.6
```

---

## HuggingFace Integration

### API Access

```go
package huggingface

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    apiToken string
    baseURL  string
}

type ModelInfo struct {
    ModelID      string                 `json:"modelId"`
    Tags         Tags                   `json:"tags"`
    Config       Config                 `json:"config"`
    CardData     CardData               `json:"cardData"`
}

type Config struct {
    Architectures        []string `json:"architectures"`
    MaxPositionEmbeddings int     `json:"max_position_embeddings"`
    NumParameters        int64    `json:"num_parameters"`
}

// FetchModelInfo retrieves model information from HuggingFace
func (c *Client) FetchModelInfo(modelID string) (*ModelInfo, error) {
    url := fmt.Sprintf("%s/api/models/%s", c.baseURL, modelID)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    if c.apiToken != "" {
        req.Header.Set("Authorization", "Bearer "+c.apiToken)
    }
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
    }
    
    var info ModelInfo
    if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
        return nil, err
    }
    
    return &info, nil
}
```

### Caching Strategy

```go
type MetadataCache struct {
    cache map[string]*ModelMetadata
    ttl   time.Duration
    mu    sync.RWMutex
}

// Get retrieves cached metadata
func (c *MetadataCache) Get(modelName string) *ModelMetadata {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if entry, ok := c.cache[modelName]; ok {
        if time.Since(entry.FetchedAt) < c.ttl {
            return entry
        }
    }
    
    return nil
}

// Set stores metadata in cache
func (c *MetadataCache) Set(modelName string, metadata *ModelMetadata) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    metadata.FetchedAt = time.Now()
    c.cache[modelName] = metadata
}
```

---

## CLI Tool for Metadata Management

```bash
# Fetch and display model metadata
llm-d-edge model info meta-llama/Llama-3.2-3B

# Generate configuration template
llm-d-edge model config meta-llama/Llama-3.2-3B > config.yaml

# Validate model configuration
llm-d-edge model validate config.yaml

# Update metadata cache
llm-d-edge model refresh
```

---

## Summary

**Recommended Approach**: HuggingFace Auto-Fetch + Manual Override

**Auto-Fetched** (from HuggingFace):
- Parameter count
- Context length
- Model family
- Architecture
- Languages
- License

**Manual Configuration** (required):
- Quality tier
- Task suitability scores
- Domain expertise scores
- Substitution rules

**Future Enhancement**: Adaptive learning from runtime performance

This hybrid approach provides:
- Quick setup (auto-fetch basics)
- Flexibility (manual fine-tuning)
- Accuracy (based on actual testing)
- Adaptability (learns over time)