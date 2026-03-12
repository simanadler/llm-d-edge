# llm-d Edge Router Architecture

## Overview

The llm-d Edge Router extends llm-d's enterprise Kubernetes infrastructure to support edge devices (laptops, mobile devices, edge servers) with intelligent routing between local and remote inference.

## Design Principles

1. **Platform Agnostic**: Support macOS, Windows, Linux, Android, and iOS through abstraction
2. **Intelligent Routing**: 6 configurable policies for optimal resource utilization
3. **Offline Capable**: Function without network connectivity when possible
4. **OpenAI Compatible**: Standard API for easy integration
5. **Production Ready**: Comprehensive testing, monitoring, and error handling

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                     Application Layer                       │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  HTTP API Server (OpenAI Compatible)                 │  │
│  │  - /v1/chat/completions                              │  │
│  │  - /v1/completions                                   │  │
│  │  - /health, /metrics                                 │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                      Routing Layer                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Router (pkg/router)                                 │  │
│  │  - Policy Evaluation                                 │  │
│  │  - Rule Matching                                     │  │
│  │  - Fallback Logic                                    │  │
│  │  - Network Monitoring                                │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                ┌───────────┴───────────┐
                │                       │
┌───────────────▼──────────┐  ┌────────▼──────────────────────┐
│   Local Inference        │  │   Remote Inference            │
│  ┌────────────────────┐  │  │  ┌─────────────────────────┐ │
│  │ Platform Engines   │  │  │  │ Remote Client           │ │
│  │ (pkg/engine)       │  │  │  │ (pkg/router)            │ │
│  └────────────────────┘  │  │  └─────────────────────────┘ │
└──────────────────────────┘  └───────────────────────────────┘
            │                              │
┌───────────▼──────────────┐  ┌───────────▼──────────────────┐
│  Platform Implementations│  │  llm-d Kubernetes Cluster    │
│  ┌────────────────────┐  │  │  ┌────────────────────────┐ │
│  │ macOS (MLX)        │  │  │  │ Gateway (Istio)        │ │
│  │ Windows (vLLM)     │  │  │  │ InferencePool          │ │
│  │ Linux (llama.cpp)  │  │  │  │ Model Servers (vLLM)   │ │
│  │ Android (llama.cpp)│  │  │  └────────────────────────┘ │
│  │ iOS (Core ML)      │  │  │                              │
│  └────────────────────┘  │  └──────────────────────────────┘
└──────────────────────────┘
```

## Core Components

### 1. HTTP API Server (`cmd/edge-router/api_server.go`)

**Responsibilities:**
- Expose OpenAI-compatible REST API
- Handle HTTP request/response lifecycle
- Provide health and metrics endpoints
- Manage graceful shutdown

**Key Features:**
- Standard OpenAI API format for easy integration
- Prometheus metrics at `/metrics`
- Health checks at `/health`
- Request logging and error handling

### 2. Router (`pkg/router/router.go`)

**Responsibilities:**
- Evaluate routing policies
- Match routing rules
- Execute routing decisions
- Handle fallbacks and retries

**Routing Policies:**
1. **local-first**: Prefer local, fallback to remote
2. **remote-first**: Prefer remote, fallback to local
3. **hybrid**: Balance based on load and capabilities
4. **cost-optimized**: Minimize cost (prefer local)
5. **latency-optimized**: Minimize latency (dynamic)
6. **mobile-optimized**: Battery and network aware

**Routing Rules:**
```yaml
routing_rules:
  - condition: "model_size > 7B"
    action: "route_remote"
  - condition: "battery_level < 20%"
    action: "route_remote"
  - condition: "network_type == 'cellular'"
    action: "prefer_local"
```

### 3. Inference Engine Interface (`pkg/engine/types.go`)

**Abstraction Layer:**
```go
type InferenceEngine interface {
    Initialize(ctx context.Context, config EngineConfig) error
    LoadModel(ctx context.Context, modelPath string) error
    Infer(ctx context.Context, request InferenceRequest) (*InferenceResponse, error)
    Unload(ctx context.Context) error
    GetCapabilities() EngineCapabilities
    GetStatistics() EngineStatistics
}
```

**Benefits:**
- Platform-agnostic interface
- Easy to add new platforms
- Consistent behavior across platforms
- Testable with mock implementations

### 4. Platform Implementations

#### macOS Engine (`internal/macos/mlx_engine.go`)

**Technology:** Apple MLX framework
**Acceleration:** Metal GPU
**Model Format:** GGUF (quantized)

**Features:**
- Automatic Metal GPU detection
- Model quantization support (Q4, Q5, Q8)
- Python script generation for MLX
- Efficient memory management

**Example:**
```go
engine := &MLXEngine{}
engine.Initialize(ctx, config)
engine.LoadModel(ctx, "~/.llm-d/models/llama-3.2-3b-q4.gguf")
response, _ := engine.Infer(ctx, request)
```

#### Windows Engine (Future: `internal/windows/vllm_engine.go`)

**Technology:** vLLM or llama.cpp
**Acceleration:** CUDA (NVIDIA) or DirectML
**Model Format:** GGUF or SafeTensors

#### Android Engine (Future: `internal/android/llama_engine.go`)

**Technology:** llama.cpp with ARM NEON
**Acceleration:** ARM NEON, Qualcomm Hexagon
**Model Format:** GGUF (quantized)

#### iOS Engine (Future: `internal/ios/coreml_engine.go`)

**Technology:** Apple Core ML
**Acceleration:** Neural Engine
**Model Format:** Core ML (.mlmodel)

### 5. Remote Client (`pkg/router/remote_client.go`)

**Responsibilities:**
- Connect to llm-d Kubernetes cluster
- Handle authentication (API keys, OAuth)
- Retry logic and error handling
- Request/response transformation

**Features:**
- Configurable timeouts and retries
- Multiple endpoint support (failover)
- Custom headers for routing
- TLS/mTLS support

### 6. Configuration (`pkg/config/config.go`)

**Format:** YAML with environment variable substitution

**Key Sections:**
- `remote`: llm-d cluster connection
- `local`: Platform-specific settings
- `routing`: Policy and rules
- `models`: Model configurations

**Example:**
```yaml
remote:
  endpoint: "${LLM_D_ENDPOINT}"
  api_key: "${LLM_D_API_KEY}"

local:
  platform: "macos"
  models_dir: "~/.llm-d/models"

routing:
  policy: "hybrid"
  fallback_to_remote: true
```

## Data Flow

### Request Flow

```
1. Client Request
   │
   ├─> HTTP API Server receives request
   │   └─> Parse and validate
   │
2. Routing Decision
   │
   ├─> Router evaluates policy
   │   ├─> Check routing rules
   │   ├─> Evaluate conditions
   │   └─> Select route (local/remote)
   │
3. Inference Execution
   │
   ├─> Local Route
   │   ├─> Check engine status
   │   ├─> Load model (if needed)
   │   ├─> Execute inference
   │   └─> Return response
   │
   └─> Remote Route
       ├─> Build HTTP request
       ├─> Send to llm-d cluster
       ├─> Handle response
       └─> Return response
   │
4. Fallback (if needed)
   │
   ├─> Primary route failed
   │   └─> Try fallback route
   │
5. Response
   │
   └─> Return to client
```

### Model Loading Flow

```
1. Model Request
   │
   ├─> Check if model loaded
   │   ├─> Yes: Use existing
   │   └─> No: Continue
   │
2. Model Discovery
   │
   ├─> Check local models directory
   │   ├─> Found: Load from disk
   │   └─> Not found: Download (future)
   │
3. Model Loading
   │
   ├─> Platform-specific loading
   │   ├─> macOS: MLX initialization
   │   ├─> Windows: vLLM/llama.cpp
   │   └─> etc.
   │
4. Model Validation
   │
   ├─> Verify model format
   ├─> Check compatibility
   └─> Test inference
   │
5. Ready
   │
   └─> Model ready for inference
```

## Routing Policies Deep Dive

### 1. Local-First Policy

**Strategy:** Maximize local inference, use remote as fallback

**Decision Logic:**
```
if local_engine.available && local_engine.can_handle(request):
    return local_inference(request)
else:
    return remote_inference(request)
```

**Use Cases:**
- Privacy-sensitive applications
- Offline-first applications
- Cost optimization

### 2. Remote-First Policy

**Strategy:** Maximize remote inference, use local as fallback

**Decision Logic:**
```
if remote_available && !offline_mode:
    return remote_inference(request)
else if local_engine.available:
    return local_inference(request)
else:
    return error
```

**Use Cases:**
- Access to larger models
- Consistent performance
- Centralized monitoring

### 3. Hybrid Policy

**Strategy:** Balance load between local and remote

**Decision Logic:**
```
score_local = calculate_score(local_latency, local_load, local_cost)
score_remote = calculate_score(remote_latency, remote_load, remote_cost)

if score_local > score_remote:
    return local_inference(request)
else:
    return remote_inference(request)
```

**Use Cases:**
- Production applications
- Variable workloads
- Optimal resource utilization

### 4. Cost-Optimized Policy

**Strategy:** Minimize inference cost

**Decision Logic:**
```
if local_engine.available:
    return local_inference(request)  # Free
else:
    return remote_inference(request)  # Paid
```

**Use Cases:**
- Budget-constrained applications
- High-volume inference
- Development/testing

### 5. Latency-Optimized Policy

**Strategy:** Minimize response latency

**Decision Logic:**
```
if local_latency < remote_latency:
    return local_inference(request)
else:
    return remote_inference(request)
```

**Use Cases:**
- Real-time applications
- Interactive chatbots
- Low-latency requirements

### 6. Mobile-Optimized Policy

**Strategy:** Optimize for battery and network

**Decision Logic:**
```
if battery_level < 20% || network_type == "cellular":
    return remote_inference(request)
else if local_engine.available:
    return local_inference(request)
else:
    return remote_inference(request)
```

**Use Cases:**
- Mobile applications
- Battery-sensitive devices
- Limited network bandwidth

## Monitoring and Observability

### Metrics

**Request Metrics:**
- `edge_router_requests_total{route="local|remote"}`
- `edge_router_request_duration_seconds{route="local|remote"}`
- `edge_router_errors_total{type="local|remote|fallback"}`

**Engine Metrics:**
- `edge_router_local_model_loaded{model="name"}`
- `edge_router_local_inference_duration_seconds`
- `edge_router_local_queue_depth`

**Remote Metrics:**
- `edge_router_remote_requests_total`
- `edge_router_remote_errors_total`
- `edge_router_remote_latency_seconds`

### Logging

**Structured Logging:**
```json
{
  "level": "info",
  "time": "2026-03-11T09:45:22Z",
  "msg": "Request routed",
  "route": "local",
  "policy": "hybrid",
  "model": "llama-3.2-3b",
  "latency_ms": 245,
  "tokens_generated": 150,
  "request_id": "req-123"
}
```

### Health Checks

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "local_engine": {
    "status": "ready",
    "model_loaded": "llama-3.2-3b",
    "platform": "macos"
  },
  "remote_endpoint": {
    "status": "reachable",
    "latency_ms": 45
  },
  "uptime_seconds": 3600
}
```

## Security

### Authentication

1. **API Key Authentication:** Bearer token in Authorization header
2. **OAuth2/OIDC:** Token-based authentication (future)
3. **mTLS:** Mutual TLS for enhanced security (future)

### Data Protection

1. **TLS/HTTPS:** All remote communication encrypted
2. **Local Model Security:** Restricted file permissions
3. **API Key Storage:** Environment variables or secret managers
4. **Request Validation:** Input sanitization and validation

### Network Security

1. **Firewall Rules:** Restrict outbound connections
2. **Certificate Validation:** Verify TLS certificates
3. **Rate Limiting:** Prevent abuse
4. **Request Signing:** Verify request integrity (future)

## Testing Strategy

### Unit Tests

**Coverage:**
- Router logic (all policies)
- Engine interface implementations
- Configuration parsing
- Network monitoring
- Metrics collection

**Example:**
```go
func TestHybridPolicy(t *testing.T) {
    router := NewRouter(config)
    request := InferenceRequest{...}
    
    route, err := router.Route(context.Background(), request)
    assert.NoError(t, err)
    assert.Equal(t, "local", route)
}
```

### Integration Tests

**Scenarios:**
- End-to-end request flow
- Fallback behavior
- Model loading and unloading
- Error handling
- Concurrent requests

### Performance Tests

**Benchmarks:**
- Routing decision latency
- Local inference throughput
- Remote client overhead
- Memory usage
- CPU utilization

## Deployment

### Standalone Binary

```bash
# Build
make build

# Run
./build/edge-router --config config.yaml --port 8080
```

### Docker Container

```bash
# Build
docker build -t llm-d-incubation/llm-d-edge:latest .

# Run
docker run -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/models:/app/models \
  llm-d-incubation/llm-d-edge:latest
```


## Future Enhancements

1. **Model Management:**
   - Automatic model download
   - Model conversion utilities
   - Model versioning

2. **Advanced Routing:**
   - A/B testing support
   - Canary deployments
   - Traffic splitting

3. **Streaming:**
   - Server-Sent Events (SSE)
   - WebSocket support
   - Chunked responses

4. **Caching:**
   - Response caching
   - Prefix caching
   - Distributed cache

5. **Multi-Model:**
   - Multiple models loaded
   - Model switching
   - Ensemble inference

6. **Observability:**
   - OpenTelemetry tracing
   - Distributed tracing
   - APM integration

## References

- [README.md](README.md): Quick start guide
- [INTEGRATION.md](INTEGRATION.md): Integration with llm-d cluster
- [config.example.yaml](config.example.yaml): Configuration reference
- [Makefile](Makefile): Build and test commands