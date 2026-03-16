# Cross-Platform Edge Device Architecture for llm-d

**Proposal for Multi-Platform Edge Device Support**

**Date**: March 11, 2026  
**Author**: Planning Mode Analysis  
**Version**: 2.0 - Cross-Platform Edition

---

## Executive Summary

This proposal extends the llm-d edge device architecture to support **multiple platforms**: macOS, Windows, Android, and iOS. The architecture uses a **platform-agnostic core** with platform-specific inference backends, enabling consistent hybrid edge-cloud inference across all major operating systems.

**Key Benefits**:
- Unified API across all platforms (desktop and mobile)
- 50%+ reduction in remote inference costs
- Platform-optimized inference engines
- Offline operation on all platforms
- Privacy-preserving local inference
- Consistent developer experience

---

## Table of Contents

1. [Cross-Platform Requirements](#cross-platform-requirements)
2. [Platform-Agnostic Architecture](#platform-agnostic-architecture)
3. [Platform-Specific Implementations](#platform-specific-implementations)
4. [Unified Component Design](#unified-component-design)
6. [Platform Comparison Matrix](#platform-comparison-matrix)
7. [Technical Specifications](#technical-specifications)
8. [Deployment Models](#deployment-models)
9. [Testing & Validation](#testing--validation)
10. [Recommendations](#recommendations)
11. [Conclusion](#conclusion)

---

## Cross-Platform Requirements

### Supported Platforms

| Platform | Priority | Target Devices | Use Cases |
|----------|----------|----------------|-----------|
| **macOS** | P0 | MacBook Pro M1-M4, Mac Studio | Development, professional use |
| **Windows** | P0 | Laptops/desktops with NVIDIA/AMD GPUs | Enterprise workstations |
| **Android** | P1 | High-end phones/tablets (8GB+ RAM) | Mobile inference, edge AI |
| **iOS** | P1 | iPhone 15 Pro+, iPad Pro | Mobile inference, privacy-focused |

### Platform-Specific Constraints

#### macOS
- **Hardware**: Apple Silicon (M1-M4) with unified memory
- **Acceleration**: Metal Performance Shaders, Neural Engine
- **Memory**: 16-64GB unified memory
- **Storage**: 256GB-2TB SSD
- **Inference Engine**: MLX, llama.cpp (Metal)

#### Windows
- **Hardware**: x86_64 with discrete GPUs (NVIDIA RTX, AMD Radeon)
- **Acceleration**: CUDA, ROCm, DirectML
- **Memory**: 16-64GB RAM + 8-24GB VRAM
- **Storage**: 512GB-2TB SSD
- **Inference Engine**: vLLM (CUDA/ROCm), llama.cpp (CUDA), ONNX Runtime

#### Android
- **Hardware**: ARM64 with NPU/GPU (Snapdragon 8 Gen 3+, Tensor G4+)
- **Acceleration**: Qualcomm Hexagon, ARM Mali, Google Tensor
- **Memory**: 8-16GB RAM
- **Storage**: 128-512GB
- **Inference Engine**: llama.cpp (Android), MediaPipe, TensorFlow Lite

#### iOS
- **Hardware**: Apple A17 Pro+, M-series with Neural Engine
- **Acceleration**: Neural Engine, Metal
- **Memory**: 8-16GB unified memory
- **Storage**: 128-1TB
- **Inference Engine**: Core ML, llama.cpp (Metal), MLX (future)

---

## Platform-Agnostic Architecture

### Core Design Principles

1. **Separation of Concerns**: Platform-agnostic routing logic + platform-specific inference
2. **Unified API**: OpenAI-compatible REST API across all platforms
3. **Pluggable Backends**: Abstract inference engine interface
4. **Consistent Configuration**: YAML-based config works on all platforms
5. **Cross-Platform SDK**: Single codebase for core logic (Go)

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    User Application                          │
│              (Same API on all platforms)                     │
└────────────────────┬────────────────────────────────────────┘
                     │ OpenAI-compatible API
┌────────────────────▼────────────────────────────────────────┐
│              llm-d Edge Router (Core)                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Routing Logic (Platform-Agnostic)                   │   │
│  │  - Request analysis                                  │   │
│  │  - Local vs. remote decision                         │   │
│  │  - Model selection                                   │   │
│  │  - Fallback handling                                 │   │
│  │  - Cost optimization                                 │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Platform Abstraction Layer                          │   │
│  │  - Inference Engine Interface                        │   │
│  │  - Model Format Adapter                              │   │
│  │  - Resource Monitor                                  │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼────────┐       ┌───────▼────────┐
│ Local Inference│       │Remote Inference│
│   (Platform-   │       │  (llm-d        │
│    Specific)   │       │   Cluster)     │
└────────────────┘       └────────────────┘
```

### Component Layers

#### Layer 1: Application Layer (Platform-Specific)
- **Desktop (macOS/Windows)**: Standalone app, CLI, system service
- **Mobile (Android/iOS)**: Native app, SDK for integration

#### Layer 2: Router Core (Platform-Agnostic)
- Written in **Go** for cross-compilation
- Implements routing logic, configuration, monitoring including [model selection](model-selection-and-confidence-architecture.md)
- Compiles to native binaries for each platform

#### Layer 3: Inference Abstraction (Platform-Agnostic Interface)
- Defines standard interface for inference engines
- Handles model format conversion
- Manages resource allocation

#### Layer 4: Inference Engines (Platform-Specific)
- **macOS**: MLX, llama.cpp (Metal)
- **Windows**: vLLM (CUDA/ROCm), llama.cpp (CUDA)
- **Android**: llama.cpp (Android), MediaPipe
- **iOS**: Core ML, llama.cpp (Metal)

---

## Platform-Specific Implementations

### macOS Implementation

**Inference Engines**:
1. **Primary**: MLX (Apple Silicon optimized)
   - Native Metal acceleration
   - Unified memory optimization
   - Best performance on M-series chips

2. **Secondary**: llama.cpp (Metal backend)
   - Broader model support
   - GGUF format
   - Fallback option

**Deployment**:
- Homebrew package: `brew install llm-d-edge`
- DMG installer for GUI app
- System service (launchd)

**Model Storage**: `~/Library/Application Support/llm-d/models/`

**Configuration**: `~/.llm-d/config.yaml`

### Windows Implementation

**Inference Engines**:
1. **Primary (NVIDIA)**: vLLM with CUDA
   - Full vLLM feature set
   - Best for RTX 3000+ series
   - Requires CUDA 12.1+

2. **Primary (AMD)**: vLLM with ROCm
   - For Radeon RX 6000+ series
   - Requires ROCm 6.0+

3. **Secondary**: llama.cpp (CUDA/ROCm)
   - Lighter weight
   - GGUF format
   - CPU fallback

4. **Tertiary**: ONNX Runtime with DirectML
   - Universal GPU support
   - Lower performance
   - Compatibility fallback

**Deployment**:
- MSI installer
- Chocolatey package: `choco install llm-d-edge`
- Windows Service
- Docker Desktop integration

**Model Storage**: `%APPDATA%\llm-d\models\`

**Configuration**: `%USERPROFILE%\.llm-d\config.yaml`

### Android Implementation

**Inference Engines**:
1. **Primary**: llama.cpp (Android NDK)
   - ARM NEON optimization
   - Qualcomm Hexagon DSP support
   - GGUF format

2. **Secondary**: MediaPipe LLM Inference
   - Google's optimized runtime
   - Tensor G-series optimization
   - Limited model support

3. **Tertiary**: TensorFlow Lite
   - Broad compatibility
   - Lower performance
   - Requires model conversion

**Deployment**:
- Android app (APK/AAB)
- SDK for app integration
- Background service

**Model Storage**: `/data/data/ai.llm-d.edge/models/`

**Configuration**: Embedded in app preferences

**Constraints**:
- Limited to quantized models (4-bit, 8-bit)
- Max model size: 4-8GB (device dependent)
- Thermal throttling management
- Battery optimization

### iOS Implementation

**Inference Engines**:
1. **Primary**: Core ML
   - Native Neural Engine acceleration
   - Best power efficiency
   - Requires Core ML model format

2. **Secondary**: llama.cpp (Metal)
   - GGUF format support
   - Good performance
   - Broader model compatibility

**Deployment**:
- iOS app (App Store)
- SDK for app integration
- Background processing

**Model Storage**: App sandbox (`Documents/models/`)

**Configuration**: Embedded in app settings

**Constraints**:
- App Store restrictions (model size limits)
- Background execution limits
- Memory pressure management
- On-device only (privacy requirement)

---

## Unified Component Design

### 1. llm-d Edge Router (Cross-Platform Core)

**Language**: Go (excellent cross-compilation) 

**Core Responsibilities**:
- Request routing logic
- Configuration management
- Monitoring and telemetry
- Model registry
- Authentication

**Platform Abstraction**:
```go
// Inference Engine Interface (platform-agnostic)
type InferenceEngine interface {
    Initialize(config EngineConfig) error
    LoadModel(modelPath string) error
    Infer(request InferenceRequest) (InferenceResponse, error)
    Unload() error
    GetCapabilities() EngineCapabilities
}

// Platform-specific implementations
type MLXEngine struct { /* macOS */ }
type VLLMEngine struct { /* Windows CUDA/ROCm */ }
type LlamaCppEngine struct { /* All platforms */ }
type CoreMLEngine struct { /* iOS */ }
type MediaPipeEngine struct { /* Android */ }
```

**Routing Decision Logic** (Platform-Agnostic):
```go
func (r *Router) RouteRequest(req Request) (Target, error) {
    // 1. Check model availability
    if !r.hasLocalModel(req.Model) {
        return RemoteTarget, nil
    }
    
    // 2. Check network connectivity
    if !r.isConnected() {
        return LocalTarget, nil
    }
    
    // 3. Apply routing policy
    switch r.config.Policy {
    case "local-first":
        if r.canServeLocally(req) {
            return LocalTarget, nil
        }
        return RemoteTarget, nil
        
    case "cost-optimized":
        localCost := r.estimateLocalCost(req)
        remoteCost := r.estimateRemoteCost(req)
        if localCost < remoteCost {
            return LocalTarget, nil
        }
        return RemoteTarget, nil
        
    case "latency-optimized":
        if req.Stream && req.PromptTokens < 2000 {
            return LocalTarget, nil // Interactive
        }
        return RemoteTarget, nil // Batch
        
    case "mobile-optimized":
        // Mobile-specific logic
        if r.isBatteryLow() || r.isThermalThrottling() {
            return RemoteTarget, nil
        }
        if req.PromptTokens < 500 {
            return LocalTarget, nil
        }
        return RemoteTarget, nil
    }
    
    return RemoteTarget, nil
}
```

### 2. Configuration System (Cross-Platform)

**Unified Configuration Format** (YAML):
```yaml
# Works on all platforms with platform-specific overrides
edge:
  platform: "auto"  # auto-detect: macos, windows, android, ios
  
  device:
    type: "auto"  # auto-detect hardware
    memory: "auto"
    storage: "auto"
  
  routing:
    policy: "hybrid"  # local-first, remote-first, hybrid, cost-optimized, mobile-optimized
    fallback: "remote"
    
  models:
    local:
      - name: "Qwen/Qwen3-0.6B"
        format: "auto"  # Platform-specific: mlx, gguf, coreml, etc.
        quantization: "4bit"
        priority: 1
      - name: "meta-llama/Llama-3.2-3B"
        format: "auto"
        quantization: "8bit"
        priority: 2
    
    remote:
      cluster_url: "https://llm-d.example.com"
      auth_token: "${LLM_D_TOKEN}"
      
  routing_rules:
    - condition: "prompt_tokens < 1000 AND model IN local_models"
      action: "route_local"
    - condition: "prompt_tokens >= 1000 OR model NOT IN local_models"
      action: "route_remote"
    - condition: "network_offline"
      action: "route_local_or_fail"
    
  # Platform-specific overrides
  platform_overrides:
    android:
      routing:
        policy: "mobile-optimized"
      models:
        local:
          - name: "Qwen/Qwen3-0.6B"
            quantization: "4bit"  # Only 4-bit on mobile
    
    ios:
      routing:
        policy: "mobile-optimized"
      models:
        local:
          - name: "Qwen/Qwen3-0.6B"
            format: "coreml"  # Prefer Core ML on iOS
            quantization: "4bit"
```

## Platform Comparison Matrix

### Feature Support Matrix

| Feature | macOS | Windows | Android | iOS | Notes |
|---------|-------|---------|---------|-----|-------|
| **Local Inference** | ✅ | ✅ | ✅ | ✅ | All platforms |
| **Remote Inference** | ✅ | ✅ | ✅ | ✅ | All platforms |
| **Offline Mode** | ✅ | ✅ | ✅ | ✅ | All platforms |
| **Streaming** | ✅ | ✅ | ✅ | ✅ | All platforms |
| **Model Download** | ✅ | ✅ | ⚠️ | ⚠️ | Mobile: WiFi only |
| **Background Inference** | ✅ | ✅ | ⚠️ | ⚠️ | Mobile: limited |
| **Multi-Model** | ✅ | ✅ | ⚠️ | ⚠️ | Mobile: 1-2 models |
| **Large Models (>7B)** | ✅ | ✅ | ❌ | ❌ | Desktop only |
| **GPU Acceleration** | ✅ | ✅ | ✅ | ✅ | Platform-specific |
| **Quantization** | 4/8-bit | 4/8-bit | 4-bit | 4-bit | Mobile: 4-bit only |

### Performance Expectations

| Platform | Model Size | Quantization | Tokens/sec | Latency (TTFT) | Memory |
|----------|------------|--------------|------------|----------------|--------|
| **macOS M4 Pro** | 3B | 4-bit | 40-60 | 200-300ms | 4GB |
| **macOS M4 Pro** | 7B | 4-bit | 25-35 | 300-500ms | 6GB |
| **Windows RTX 4090** | 3B | 4-bit | 80-120 | 100-200ms | 4GB |
| **Windows RTX 4090** | 7B | 4-bit | 50-70 | 200-300ms | 6GB |
| **Android (SD 8 Gen 3)** | 0.5B | 4-bit | 15-25 | 500-800ms | 2GB |
| **Android (SD 8 Gen 3)** | 3B | 4-bit | 5-10 | 1-2s | 4GB |
| **iOS (A17 Pro)** | 0.5B | 4-bit | 20-30 | 400-600ms | 2GB |
| **iOS (A17 Pro)** | 3B | 4-bit | 8-12 | 800ms-1.5s | 4GB |

---

## Technical Specifications

### API Specification (All Platforms)

**Endpoint**: `POST /v1/chat/completions` (OpenAI-compatible)

**Request**:
```json
{
  "model": "Qwen/Qwen3-0.6B",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "stream": true,
  "max_tokens": 100,
  "temperature": 0.7,
  "llm_d_routing": {
    "policy": "local-first",
    "max_local_latency_ms": 5000,
    "allow_fallback": true
  }
}
```

**Response**:
```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "Qwen/Qwen3-0.6B",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello! How can I help you?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 5,
    "completion_tokens": 7,
    "total_tokens": 12
  },
  "llm_d_metadata": {
    "routing_target": "local",
    "inference_engine": "mlx",
    "latency_ms": 234,
    "platform": "macos"
  }
}
```

### Resource Requirements by Platform

#### macOS
- **Minimum**: M1, 16GB RAM, 50GB storage
- **Recommended**: M3/M4, 32GB RAM, 100GB storage
- **Optimal**: M4 Pro/Max, 64GB RAM, 500GB storage

#### Windows
- **Minimum**: RTX 3060 (12GB), 16GB RAM, 100GB storage
- **Recommended**: RTX 4070 (12GB), 32GB RAM, 250GB storage
- **Optimal**: RTX 4090 (24GB), 64GB RAM, 500GB storage

#### Android
- **Minimum**: Snapdragon 8 Gen 2, 8GB RAM, 128GB storage
- **Recommended**: Snapdragon 8 Gen 3, 12GB RAM, 256GB storage
- **Optimal**: Snapdragon 8 Gen 4, 16GB RAM, 512GB storage

#### iOS
- **Minimum**: A16 Bionic, 6GB RAM, 128GB storage
- **Recommended**: A17 Pro, 8GB RAM, 256GB storage
- **Optimal**: M2/M4 (iPad), 16GB RAM, 512GB storage

---

## Deployment Models

### Desktop Deployment

#### macOS
```bash
# Homebrew installation
brew install llm-d-edge

# Initialize
llm-d-edge init

# Start service
brew services start llm-d-edge

# Or run as app
open /Applications/llm-d-edge.app
```

#### Windows
```powershell
# Chocolatey installation
choco install llm-d-edge

# Initialize
llm-d-edge init

# Start service
Start-Service llm-d-edge

# Or run as app
Start-Process "C:\Program Files\llm-d-edge\llm-d-edge.exe"
```

### Mobile Deployment

#### Android
```kotlin
// SDK integration
dependencies {
    implementation 'ai.llm-d:edge-sdk:1.0.0'
}

// Initialize
val llmDEdge = LLMDEdge.Builder(context)
    .setClusterUrl("https://llm-d.example.com")
    .setAuthToken(token)
    .setRoutingPolicy(RoutingPolicy.MOBILE_OPTIMIZED)
    .build()

// Inference
llmDEdge.chat(
    model = "Qwen/Qwen3-0.6B",
    messages = listOf(Message("user", "Hello!")),
    onResponse = { response -> /* handle */ }
)
```

#### iOS
```swift
// SDK integration
import LLMDEdge

// Initialize
let llmDEdge = LLMDEdge.Builder()
    .setClusterURL("https://llm-d.example.com")
    .setAuthToken(token)
    .setRoutingPolicy(.mobileOptimized)
    .build()

// Inference
llmDEdge.chat(
    model: "Qwen/Qwen3-0.6B",
    messages: [Message(role: "user", content: "Hello!")],
    onResponse: { response in /* handle */ }
)
```

---

## Testing & Validation

### Cross-Platform Testing Strategy

#### Unit Tests (Platform-Agnostic)
- Routing logic
- Configuration parsing
- Model format detection
- API compatibility

#### Integration Tests (Platform-Specific)
- Inference engine integration
- Model loading/unloading
- Memory management
- Resource monitoring

#### Performance Tests
- Latency benchmarks (per platform)
- Throughput tests
- Memory usage profiling
- Battery impact (mobile)
- Thermal behavior (mobile)

#### User Acceptance Tests
- Installation experience (<10 minutes)
- Model download and conversion
- Routing policy configuration
- App integration (mobile)

### Platform-Specific Test Matrices

#### macOS
- M1, M2, M3, M4 (all variants)
- macOS 13, 14, 15
- 16GB, 32GB, 64GB RAM

#### Windows
- NVIDIA RTX 3000, 4000 series
- AMD Radeon RX 6000, 7000 series
- Windows 10, 11
- 16GB, 32GB, 64GB RAM

#### Android
- Snapdragon 8 Gen 2, 3, 4
- Google Tensor G3, G4
- Android 13, 14, 15
- 8GB, 12GB, 16GB RAM

#### iOS
- iPhone 14 Pro, 15 Pro, 16 Pro
- iPad Pro (M2, M4)
- iOS 17, 18
- 6GB, 8GB, 16GB RAM

## Recommendations

### Best Architectural Approach

**Hybrid Edge-Cloud with Platform Abstraction**:
1. **Core router in Go** for cross-compilation
2. **Platform-specific inference engines** for optimal performance
3. **Unified API** for consistent developer experience
4. **Graceful degradation** for offline operation
5. **Mobile-optimized routing** for battery/thermal management


### Key Success Factors

1. **Platform Abstraction**: Clean separation between core logic and platform-specific code
2. **Performance**: Platform-optimized inference engines
3. **Developer Experience**: Simple installation and integration
4. **Mobile Optimization**: Battery and thermal management
5. **Consistent API**: Same interface across all platforms

### Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Platform fragmentation | Strong abstraction layer, comprehensive testing |
| Performance variance | Platform-specific optimization, benchmarking |
| Mobile constraints | Quantization, thermal management, background limits |
| App Store restrictions | Core ML compliance, size limits, privacy policies |
| Maintenance burden | Automated testing, CI/CD, community contributions |

---

## Conclusion

This cross-platform architecture enables llm-d to support edge devices across all major platforms (macOS, Windows, Android, iOS) while maintaining:

- **Consistent API**: Same OpenAI-compatible interface everywhere
- **Optimal Performance**: Platform-specific inference engines
- **Unified Experience**: Single configuration format, consistent behavior
- **Future-Proof**: Extensible to new platforms and inference engines

The phased approach (desktop first, then mobile) allows for iterative development and validation while building toward comprehensive cross-platform support.

**Next Steps**:
1. Review and approve this proposal
2. Begin Phase 1 implementation (desktop platforms)
3. Establish cross-platform CI/CD pipeline
4. Create platform-specific working groups
5. Engage with platform-specific communities (Metal, CUDA, Android NDK, Core ML)