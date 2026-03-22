# llm-d Edge Router

Hybrid edge-cloud inference routing for llm-d, enabling intelligent routing between local edge device inference and remote cluster inference.

## Features

- **Intelligent Routing**: Automatically routes requests between local and remote inference based on configurable policies
- **Cross-Platform**: Supports macOS, Windows, Android, and iOS with platform-specific optimizations
- **Multiple Routing Policies**: Local-first, remote-first, hybrid, cost-optimized, latency-optimized, and mobile-optimized
- **Graceful Degradation**: Automatic fallback when primary target fails
- **Offline Support**: Works offline with local models
- **OpenAI-Compatible API**: Drop-in replacement for OpenAI API
- **Platform-Optimized Engines**:
  - **macOS**: MLX (Metal acceleration)
  - **Windows**: vLLM (CUDA/ROCm) or llama.cpp
  - **Android**: llama.cpp (ARM NEON + Hexagon DSP)
  - **iOS**: Core ML or llama.cpp (Metal)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    User Application                          │
│              (OpenAI-compatible API)                         │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│              llm-d Edge Router (Core)                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Routing Logic (Platform-Agnostic)                   │   │
│  │  - Request analysis                                  │   │
│  │  - Local vs. remote decision                         │   │
│  │  - Fallback handling                                 │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Platform Abstraction Layer                          │   │
│  │  - Inference Engine Interface                        │   │
│  │  - Model Format Adapter                              │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼────────┐       ┌───────▼────────┐
│ Local Inference│       │Remote Inference│
│   (MLX, vLLM,  │       │  (llm-d        │
│   llama.cpp)   │       │   Cluster)     │
└────────────────┘       └────────────────┘
```

## Quick Start

### Prerequisites

#### macOS
```bash
# Install MLX
pip3 install mlx-lm

# Install Go 1.22+
brew install go
```

#### Windows
```bash
# Install Go 1.22+
# Download from https://go.dev/dl/

# Install CUDA (for NVIDIA GPUs)
# Or ROCm (for AMD GPUs)
```

### Installation

```bash
# Clone the repository
git clone https://github.com/llm-d-incubation/llm-d-edge.git
cd llm-d-edge/edge-router

# Build using Makefile (recommended - builds to build/edge-router)
make build

# Or build manually
go build -o build/edge-router ./cmd/edge-router

# Or install to $GOPATH/bin (typically ~/go/bin)
go install ./cmd/edge-router
```

### Configuration

1. Copy the example configuration:
```bash
cp config.example.yaml config.yaml
```

2. Edit `config.yaml` to configure your models and remote cluster:
```yaml
edge:
  platform: auto
  routing:
    policy: hybrid
    fallback: remote
  models:
    local:
      - name: "Qwen/Qwen3-0.6B"
        format: auto
        quantization: "4bit"
    remote:
      cluster_url: "https://your-llm-d-cluster.com"
      auth_token: "${LLMD_AUTH_TOKEN}"
```

3. Set environment variables:
```bash
export LLMD_AUTH_TOKEN="your-auth-token"
```

### Running

```bash
# If you built with make or go build
./build/edge-router --config config.yaml --port 8080

# Or with custom log level
./build/edge-router --config config.yaml --port 8080 --log-level debug

# If you installed with go install (and ~/go/bin is in your PATH)
edge-router --config config.yaml --port 8080

# Or use the Makefile
make run
```

### Usage

The edge router provides an OpenAI-compatible API:

```bash
# Chat completions
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen/Qwen3-0.6B",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "max_tokens": 100,
    "temperature": 0.7
  }'

# Text completions
curl -X POST http://localhost:8080/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Qwen/Qwen3-0.6B",
    "prompt": "Once upon a time",
    "max_tokens": 100
  }'

# Health check
curl http://localhost:8080/health

# List models
curl http://localhost:8080/v1/models

# Metrics
curl http://localhost:8080/metrics
```

## Routing Policies

### Local-First
Prefers local inference when possible. Falls back to remote if local capacity is exceeded.

```yaml
routing:
  policy: local-first
  fallback: remote
```

### Remote-First
Prefers remote cluster inference. Falls back to local if remote is unavailable.

```yaml
routing:
  policy: remote-first
  fallback: local
```

### Hybrid
Balances between local and remote based on request characteristics:
- Small requests (<1000 tokens): Local
- Large requests (≥1000 tokens): Remote

```yaml
routing:
  policy: hybrid
  fallback: remote
```

### Cost-Optimized
Minimizes cost by preferring local inference (essentially free) over remote (metered).

```yaml
routing:
  policy: cost-optimized
  fallback: remote
```

### Latency-Optimized
Minimizes latency:
- Interactive/streaming requests: Local
- Batch requests: Remote (better throughput)

```yaml
routing:
  policy: latency-optimized
  fallback: remote
```

### Mobile-Optimized
Optimizes for mobile constraints:
- Low battery: Remote
- Thermal throttling: Remote
- Small requests: Local
- Default: Remote

```yaml
routing:
  policy: mobile-optimized
  fallback: remote
```

## Custom Routing Rules

Define custom routing rules that override the policy:

```yaml
routing_rules:
  # Always use local for small requests
  - condition: "prompt_tokens < 500"
    action: "route_local"

  # Always use remote for specific models
  - condition: "model == 'gpt-4'"
    action: "route_remote"

  # Offline mode
  - condition: "network_offline"
    action: "route_local_or_fail"
```

## Using Model Manager for Local Models

The [model-manager](../model-manager) tool can automatically install models and generate a YAML configuration file that you can use to populate your edge-router configuration.

### Workflow

1. **Install models using model-manager**:
```bash
cd ../model-manager
./model-manager select --tasks code,chat
```

2. **Review the generated configuration**:
```bash
# Default location (macOS)
cat ~/Library/Application\ Support/llm-d/installed-models.yaml

# Or custom location if specified
cat /path/to/installed-models.yaml
```

3. **Copy models to your edge-router config**:

The generated file contains complete model configurations with capabilities and matching rules:

```yaml
edge:
  models:
    local:
      - name: "Qwen/Qwen2.5-3B-Instruct"
        format: "mlx"
        quantization: "4bit"
        priority: 1
        path: "/Users/you/Library/Application Support/llm-d/models/Qwen--Qwen2.5-3B-Instruct"
        
        capabilities:
          parameter_count: "3B"
          context_length: 8192
          model_family: "qwen"
          quality_tier: "medium"
          
          tasks:
            chat: 0.9
            code: 0.7
          
          domains:
            general: 0.9
            technical: 0.7
        
        matching:
          can_substitute:
            - pattern: "*3b*"
            - pattern: "qwen*"
          
          exclude_patterns:
            - "gpt-4*"
```

4. **Copy the models you want into your `config.yaml`**:

Simply copy the model entries from `installed-models.yaml` into your edge-router `config.yaml` under `edge.models.local`.

### Benefits

- **Automatic capability detection**: Model-manager detects and scores model capabilities
- **Matching rules**: Automatically generates substitution patterns based on model characteristics
- **Consistent structure**: Generated YAML matches edge-router's expected format
- **Easy updates**: Regenerate the file whenever you install/uninstall models

See [model-manager README](../model-manager/README.md) for more details on model installation and management.

## Platform-Specific Configuration

### macOS (MLX)

```yaml
platform_overrides:
  macos:
    models:
      local:
        - name: "Qwen/Qwen3-0.6B"
          format: "mlx"
          quantization: "4bit"
```

**Model Preparation**:
```bash
# Convert HuggingFace model to MLX format
python -m mlx_lm.convert \
  --hf-path Qwen/Qwen3-0.6B \
  --mlx-path ~/Library/Application\ Support/llm-d/models/Qwen--Qwen3-0.6B \
  --quantize \
  --q-bits 4
```

### Windows (vLLM/llama.cpp)

```yaml
platform_overrides:
  windows:
    models:
      local:
        - name: "Qwen/Qwen3-0.6B"
          format: "gguf"
          quantization: "4bit"
```

### Android

```yaml
platform_overrides:
  android:
    routing:
      policy: "mobile-optimized"
    models:
      local:
        - name: "Qwen/Qwen3-0.6B"
          format: "gguf"
          quantization: "4bit"
```

### iOS

```yaml
platform_overrides:
  ios:
    routing:
      policy: "mobile-optimized"
    models:
      local:
        - name: "Qwen/Qwen3-0.6B"
          format: "coreml"
          quantization: "4bit"
```

## Development

### Project Structure

```
edge-router/
├── cmd/
│   └── edge-router/          # Main application
│       ├── main.go           # Entry point
│       └── api_server.go     # HTTP API server
├── pkg/
│   ├── config/               # Configuration management
│   ├── engine/               # Inference engine interface
│   ├── router/               # Routing logic
│   └── models/               # Model management
├── internal/
│   ├── macos/                # macOS-specific (MLX)
│   ├── windows/              # Windows-specific (vLLM/llama.cpp)
│   ├── android/              # Android-specific
│   └── ios/                  # iOS-specific
└── config.example.yaml       # Example configuration
```

### Building for Different Platforms

```bash
# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o edge-router-macos-arm64 ./cmd/edge-router

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o edge-router-macos-amd64 ./cmd/edge-router

# Windows
GOOS=windows GOARCH=amd64 go build -o edge-router-windows.exe ./cmd/edge-router

# Linux
GOOS=linux GOARCH=amd64 go build -o edge-router-linux ./cmd/edge-router
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/router/...
```

## Performance

### Expected Performance

| Platform | Model Size | Quantization | Tokens/sec | Latency (TTFT) | Memory |
|----------|------------|--------------|------------|----------------|--------|
| **macOS M4 Pro** | 3B | 4-bit | 40-60 | 200-300ms | 4GB |
| **macOS M4 Pro** | 7B | 4-bit | 25-35 | 300-500ms | 6GB |
| **Windows RTX 4090** | 3B | 4-bit | 80-120 | 100-200ms | 4GB |
| **Windows RTX 4090** | 7B | 4-bit | 50-70 | 200-300ms | 6GB |
| **Android (SD 8 Gen 3)** | 0.5B | 4-bit | 15-25 | 500-800ms | 2GB |
| **iOS (A17 Pro)** | 0.5B | 4-bit | 20-30 | 400-600ms | 2GB |

### Routing Overhead

- Routing decision: <10ms
- Network check: <5ms
- Total overhead: <100ms

## Troubleshooting

### MLX Not Found (macOS)

```bash
# Install MLX
pip3 install mlx-lm

# Verify installation
python3 -c "import mlx_lm; print('MLX installed')"
```

### Model Not Found

```bash
# Check model path
ls ~/Library/Application\ Support/llm-d/models/

# Download and convert model
python -m mlx_lm.convert --hf-path Qwen/Qwen3-0.6B --mlx-path ~/Library/Application\ Support/llm-d/models/Qwen--Qwen3-0.6B
```

### Remote Cluster Connection Failed

```bash
# Check network connectivity
curl https://your-llm-d-cluster.com/health

# Verify auth token
echo $LLMD_AUTH_TOKEN

# Test with explicit token
curl -H "Authorization: Bearer $LLMD_AUTH_TOKEN" https://your-llm-d-cluster.com/v1/models
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0. See [LICENSE](../LICENSE) for details.

## Related Projects

- [llm-d](https://github.com/llm-d/llm-d) - Main llm-d project
- [MLX](https://github.com/ml-explore/mlx) - Apple's ML framework
- [vLLM](https://github.com/vllm-project/vllm) - High-performance LLM inference
- [llama.cpp](https://github.com/ggerganov/llama.cpp) - C++ LLM inference

## Support

- [Documentation](https://llm-d.ai/docs/edge-router)
- [Slack](https://llm-d.ai/slack)
- [GitHub Issues](https://github.com/llm-d-incubation/llm-d-edge/issues)