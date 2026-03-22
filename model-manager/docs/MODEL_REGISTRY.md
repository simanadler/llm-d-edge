# Model Registry Architecture

## Overview

The Model Manager uses a flexible registry architecture that supports multiple model sources. This allows organizations to use HuggingFace, enterprise model repositories, or custom registries.

## Architecture

### ModelRegistry Interface

```go
type ModelRegistry interface {
    GetModels(ctx context.Context) ([]types.ModelMetadata, error)
    SearchModels(ctx context.Context, query string, limit int) ([]types.ModelMetadata, error)
    GetModel(ctx context.Context, modelID string) (*types.ModelMetadata, error)
    Name() string
}
```

### Composite Registry

The `CompositeRegistry` combines multiple registries:

```go
registry := NewCompositeRegistry(
    NewHuggingFaceRegistry(),
    NewEnterpriseRegistry("https://models.company.com"),
    NewLocalRegistry("/path/to/models"),
)
```

Models are deduplicated across registries, with the first registry taking precedence.

## Built-in Registries

### 1. HuggingFace Registry

**Purpose**: Discover popular open-source models from HuggingFace

**Features**:
- Curated list of popular models across size ranges (0.5B to 72B)
- Automatic metadata extraction
- License detection
- Task capability inference

**Models Included**:
- **Small (0.5B-1.7B)**: Qwen2.5-0.5B, SmolLM-1.7B
- **Medium (3B)**: Qwen2.5-3B, Llama-3.2-3B, Phi-3-mini, StableLM-3B
- **Large (7-8B)**: Mistral-7B, Llama-3.1-8B, Qwen2.5-7B
- **Very Large (14B+)**: Qwen2.5-14B, Qwen2.5-32B, Qwen2.5-72B, Llama-3.1-70B, Mixtral-8x7B

**Usage**:
```go
hfRegistry := NewHuggingFaceRegistry()
models, err := hfRegistry.GetModels(ctx)
```

### 2. Enterprise Registry (Planned)

**Purpose**: Connect to private model repositories

**Features**:
- Authentication support
- Custom model metadata
- Policy enforcement
- Approval workflows

**Example**:
```go
enterpriseRegistry := NewEnterpriseRegistry(EnterpriseConfig{
    URL: "https://models.company.com",
    APIKey: os.Getenv("MODEL_REGISTRY_KEY"),
    TLSConfig: tlsConfig,
})
```

### 3. Local Registry (Planned)

**Purpose**: Discover models installed locally

**Features**:
- Scan local directories
- Detect model formats (MLX, GGUF, SafeTensors)
- Extract metadata from model files
- Track usage statistics

**Example**:
```go
localRegistry := NewLocalRegistry(LocalConfig{
    ModelsDir: "/Users/username/.cache/models",
    ScanRecursive: true,
})
```

## Registry Selection Strategy

The `CompositeRegistry` queries registries in order and combines results:

1. **Query all registries** in parallel
2. **Deduplicate** by model ID or name
3. **Merge metadata** (first registry wins for conflicts)
4. **Return combined list**

## Adding Custom Registries

To add a custom registry, implement the `ModelRegistry` interface:

```go
type MyCustomRegistry struct {
    apiClient *http.Client
    baseURL   string
}

func (r *MyCustomRegistry) GetModels(ctx context.Context) ([]types.ModelMetadata, error) {
    // Query your API
    resp, err := r.apiClient.Get(r.baseURL + "/models")
    // ... parse and return models
}

func (r *MyCustomRegistry) SearchModels(ctx context.Context, query string, limit int) ([]types.ModelMetadata, error) {
    // Implement search
}

func (r *MyCustomRegistry) GetModel(ctx context.Context, modelID string) (*types.ModelMetadata, error) {
    // Get specific model
}

func (r *MyCustomRegistry) Name() string {
    return "my-custom-registry"
}
```

Then use it:

```go
matcher := NewModelMatcherWithRegistry(
    NewCompositeRegistry(
        NewMyCustomRegistry(),
        NewHuggingFaceRegistry(),
    ),
)
```

## Configuration

Future versions will support registry configuration via YAML:

```yaml
registries:
  - type: huggingface
    enabled: true
    
  - type: enterprise
    enabled: true
    url: https://models.company.com
    api_key_env: MODEL_REGISTRY_KEY
    
  - type: local
    enabled: true
    models_dir: ~/.cache/models
    scan_recursive: true
```

## Model Metadata

Each registry provides standardized metadata:

```go
type ModelMetadata struct {
    Name            string             // Model name
    ParameterCount  string             // "3B", "7B", etc.
    ContextLength   int                // Token context window
    ModelFamily     string             // "llama", "mistral", etc.
    QualityTier     string             // "base", "instruct", "chat"
    Tasks           map[string]float64 // Task capabilities
    Domains         map[string]float64 // Domain expertise
    License         string             // License type
    DownloadSizeGB  float64            // Download size
    Formats         []string           // Available formats
    Quantizations   []string           // Available quantizations
    HuggingFaceRepo string             // HF repo ID (if applicable)
}
```

## Benefits

### 1. Flexibility
- Use multiple model sources
- Easy to add new registries
- No vendor lock-in

### 2. Enterprise-Ready
- Support private model repositories
- Enforce organizational policies
- Track model usage and compliance

### 3. Extensibility
- Simple interface to implement
- Plug-and-play architecture
- Custom metadata support

### 4. Offline Support
- Local registry works without internet
- Cached metadata
- Fallback mechanisms

## Future Enhancements

### Phase 2
- [ ] Enterprise registry implementation
- [ ] Local registry with model scanning
- [ ] Configuration file support
- [ ] Registry caching

### Phase 3
- [ ] Model versioning support
- [ ] Automatic updates
- [ ] Registry health monitoring
- [ ] Model popularity tracking

### Phase 4
- [ ] Federated registries
- [ ] Model recommendation based on usage
- [ ] A/B testing support
- [ ] Cost optimization

## Examples

### Basic Usage

```go
// Use default HuggingFace registry
matcher := NewModelMatcher()
models, err := matcher.FindCandidates(ctx, deviceProfile)
```

### Multiple Registries

```go
// Combine multiple sources
registry := NewCompositeRegistry(
    NewEnterpriseRegistry(enterpriseConfig),
    NewHuggingFaceRegistry(),
    NewLocalRegistry(localConfig),
)

matcher := NewModelMatcherWithRegistry(registry)
models, err := matcher.FindCandidates(ctx, deviceProfile)
```

### Custom Registry

```go
// Use only your custom registry
myRegistry := NewMyCustomRegistry(config)
matcher := NewModelMatcherWithRegistry(myRegistry)
models, err := matcher.FindCandidates(ctx, deviceProfile)
```

## See Also

- [Advanced Model Manager Requirements](../../docs/advanced-model-manager-requirements.md)
- [Cross-Platform Architecture](../../docs/advanced-model-manager-cross-platform-architecture.md)
- [Implementation Summary](../IMPLEMENTATION.md)