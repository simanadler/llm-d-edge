# llm-d-edge Proposal

**Proposal for llm-d Edge Support**

**Date**: March 11, 2026  
**Author**: Sima Nadler 
**Version**: 1.0 


## Table of Contents

- [Executive Summary](#executive-summary)
- [Current State Analysis](#current-state-analysis)
- [Requirements & Constraints](#requirements--constraints)
- [Proposed Edge Platform-Agnostic Architecture](#proposed-edge-platform-agnostic-architecture)
- [Existing Solutions](#existing-solutions)
- [Critical Gap: Hardware-Aware Routing](#critical-gap-hardware-aware-routing)
- [Our Proposal's Unique Position](#our-edge-routers-unique-position)
- [Strategic Positioning](#strategic-positioning)
- [Conclusion](#conclusion)
- [References](#references)

---

## Executive Summary

This is a proposal to extend llm-d such that inferencing may be done on models running on edge devices or on the enterprise's centralized inferencing capabilities.  The idea is to leverage edge devices such as laptops, mobile devices, and in the future IoT devices, enabling enterprises to take advantage of the compute power of their employees, customers, and supplier devices in addition to their data center running llm-d. The solution introduces a **Hybrid Edge-Cloud Architecture** where inference requests coming from edge devices are routed either to local models on the device or to remote cluster resources.

**Key Benefits to Enterprise**:
- Substantial reduction in enterprise's inference costs
- Leverage compute power of enterprise provided personal devices, which are currently under-utilized
- Ensure privacy of sensitive enterprise data and personal data

**Key Benefits to Users**:
- Save on budget for expensive LLMs 
- Low latency! <100ms routing overhead
- Offline inferencing capability
- Privacy-preserving (local data stays local)
- No need to manually determine whether to use a local vs a remote model
- No need to look for and manually download models appropriate for the device

## llm-d Current State


llm-d is a **Kubernetes-native distributed inference serving stack** designed for datacenter environments with:

**Core Components**:
- **vLLM Model Servers**: Run on datacenter accelerators (NVIDIA GPUs, AMD GPUs, Intel XPU/HPU, Google TPUs)
- **Inference Gateway (IGW)**: Intelligent request routing and load balancing within the data center
- **Inference Scheduler**: Prefix-cache-aware, latency-aware, and load-aware balancing
- **Kubernetes Orchestration**: Scaling, resource management, and workload control

**Key Features**:
- Disaggregated serving (prefill/decode separation)
- Prefix cache hierarchy (HBM, host memory, remote storage)
- High-performance networking (RDMA, NVLink, TPU ICI)
- Multi-accelerator support
- Variant autoscaling

**Current Scope**:
- Datacenter-class accelerators (A100+, MI250+, TPU v5e+, XPU)
- Kubernetes 1.29+ clusters (10-100k nodes)
- High-speed interconnects (100-16,000 Gbps)
- Large models (1B+ parameters)

### Gap Analysis

**What's Missing in llm-d for Edge Devices**:

| Missing Feature | Status |
|-----------------|--------|
| Client-side routing logic | ❌ Not Available |
| Support for Apple Silicon (M1/M2/M3/M4) | ❌ Not Available |
| Edge-optimized model formats (MLX, GGUF) | ❌ Not Available |
| Standalone inference outside Kubernetes | ❌ Not Available |
| Hybrid local/remote routing | ❌ Not Available |
| Edge device management | ❌ Not Available |

---

## Requirements & Constraints

### Functional Requirements

1. **User-Local Models**: Edge devices run models exclusively for the local user
2. **Intelligent Routing**: Client-side logic decides between local and remote inference
3. **Offline Operation**: System works without network connectivity using local models when necessary
4. **Consistent API**: OpenAI-compatible API for both local and remote inference
5. **Graceful Degradation**: Automatic fallback to remote when local inference fails
6. **(Future) Model Management**: Download, convert, and manage models on edge devices
7. **(Future) Confidence Based Routing**: Inference Result Confidence Scoring based re-routing strategies

### Non-Functional Requirements

1. **Performance**: <100ms routing decision overhead
2. **Reliability**: 99.9% uptime for local inference
3. **Latency**: Comparable to remote inference for small models
4. **Storage**: Support 50-500GB model storage
5. **Memory**: Efficient use of 16-64GB unified memory
6. **Developer Experience**: <10 minute setup time

### Constraints

1. **No Cluster Integration**: Edge devices don't join Kubernetes cluster as nodes
2. **Privacy**: User data and models stay on edge device
3. **No Cluster Changes**: Preserve existing llm-d functionality
4. **Single User**: No multi-user support on edge devices - at least initially


**Future Enhancements**: See thoughts about [future work](future-work.md)).


## Proposed Edge Platform-Agnostic Architecture

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
│  │  - Model choice decision (local / remote different)  │   │
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
- Implements routing logic, configuration, monitoring
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


## Existing Solutions
Several tools and systems automatically route inference requests to the most appropriate model. This section provides a comprehensive comparison of existing solutions and how our edge-router project differs.

Most systems route based on:

### 1. Query Complexity
- Token count
- Semantic complexity
- Task type (classification vs. generation)

### 2. Performance Requirements
- Latency constraints
- Quality thresholds
- Cost budgets

### 3. Model Capabilities
- Context window size
- Specialized skills (code, math, etc.)
- Language support

### 4. Operational Factors
- Model availability
- Current load
- Rate limits

### 1. LiteLLM

**Type:** Open-source  
**Focus:** Cloud-based multi-provider routing

**Key Features:**
- Automatic fallback routing - if one model fails, routes to alternatives
- Load balancing across multiple deployments
- Cost-based routing - routes to cheaper models when appropriate
- Latency-based routing - selects fastest available model
- Unified API interface (OpenAI-compatible)
- Supports 100+ LLM providers

### 2. OpenRouter

**Type:** Commercial service  
**Focus:** Unified API for multiple model providers

**Key Features:**
- Routes requests to the best model based on your criteria
- Provides access to multiple model providers through a single API
- Offers automatic fallback and load balancing
- Includes cost optimization features

### 3. Martian

**Type:** Commercial service
**Focus:** Performance optimization and testing

**Key Features:**
- Model routing based on performance metrics
- A/B testing capabilities
- Cost and latency optimization
- Request/response caching

### 4. RouteLLM (Berkeley Research)

**Type:** Open-source research framework
**Focus:** Cost-performance optimization

**Key Features:**
- Routes simple queries to cheaper models, complex ones to expensive models
- Uses a trained router model to classify query complexity
- Can reduce costs by 85% while maintaining 95% of GPT-4 performance
- ML-based routing decisions

### 5. Portkey

**Type:** Open-source (MIT License)
**Focus:** Enterprise-grade routing and monitoring
**Repository:** [Portkey-AI/gateway](https://github.com/Portkey-AI/gateway)

**Key Features:**
- Routes to 200+ LLMs with integrated guardrails
- Intelligent routing based on custom rules
- Automatic retries and fallbacks
- Load balancing across providers
- Performance monitoring and analytics
- 50+ AI guardrails

### 6. BentoML

**Type:** Open-source platform
**Focus:** Model serving and orchestration

**Key Features:**
- Model serving with intelligent routing
- Traffic splitting for A/B testing
- Adaptive batching
- Multi-model orchestration

### 7. Claude Code Router

**Type:** Open-source
**Focus:** Claude Code integration with multi-provider routing
**Repository:** [musistudio/claude-code-router](https://github.com/musistudio/claude-code-router)

**Key Features:**
- Routes Claude Code requests to different models based on task type (background tasks, thinking, long context, web search)
- Multi-provider support including **local models via Ollama** and cloud providers (OpenRouter, DeepSeek, Gemini, Volcengine, SiliconFlow)
- Request/response transformation for different providers
- Dynamic model switching via `/model` command
- CLI model management
- GitHub Actions integration
- Plugin system for custom transformers
- Configuration-based routing rules

**Use Case:** Specifically designed for developers using Claude Code (Anthropic's coding assistant), allowing them to route different types of coding tasks to appropriate models (both local and cloud) while maintaining Claude Code's interface.

**Local Model Support:** Can route to Ollama for local execution (e.g., `qwen2.5-coder:latest` for background tasks), but routing decision is configuration-based rather than hardware-aware.


## Critical Gap: Hardware-Aware Routing

**Most existing solutions require manual configuration for local vs. remote routing and don't automatically detect hardware capabilities.**

### Comparison Matrix:

| Feature | LiteLLM | OpenRouter | Martian | RouteLLM | Portkey | BentoML | Claude Code Router | Our Edge Router |
|---------|---------|------------|---------|----------|---------|---------|-------------------|-----------------|
| Cloud-to-cloud routing | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Local model support | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ (Ollama) | ✅ |
| Cost optimization | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| Latency-based routing | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ | ❌ | ✅ |
| Task-based routing | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ | ✅ |
| Configuration-based routing | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Hardware capability detection** | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Automatic local/remote decision** | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Platform-specific optimization** | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Edge-first architecture** | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |

### Key Distinctions:

**Cloud-Only Tools (LiteLLM, OpenRouter, Martian, Portkey):**
- All models accessed via API endpoints
- Routing decisions based on cost, latency, and model capabilities
- No awareness of client-side hardware (RAM, GPU, CPU)
- No local execution support

**Hybrid Tools with Manual Configuration:**
- **BentoML** - Can serve local models but requires manual deployment configuration
- **Claude Code Router** - Routes to local Ollama OR cloud providers, but user must manually configure which tasks use which models (e.g., "background tasks → Ollama qwen2.5-coder", "thinking → DeepSeek cloud")
- **RouteLLM** - Routes based on query complexity but assumes all models are cloud-based

**Local-Only Tools:**
- **Ollama** - Runs models locally but doesn't route between local/remote based on hardware
- **LocalAI** - Local inference server, but no automatic hardware-aware routing or cloud fallback
- **LM Studio** - Local model runner with hardware detection, but no cloud fallback routing
- **Jan.ai** - Local-first with hardware awareness, but limited routing logic and no intelligent cloud fallback


## Strategic Positioning

### Key Differentiators of Our Proposal:

1. **Hardware Capability Detection**
   - Detects platform capabilities (macOS MLX support via `mlx_engine.go`)
   - Assesses available RAM, GPU, and CPU resources
   - Platform-specific optimization (Android, iOS, Windows, macOS)

2. **Automatic Local/Remote Routing**
   - Determines if device can run model locally
   - Automatically falls back to cloud are alternative local model when needed
   - Configuration-based routing rules

3. **Edge-First Architecture**
   - Designed for resource-constrained devices
   - Minimizes latency by preferring local execution
   - Reduces costs by avoiding unnecessary cloud calls

4. **Platform-Specific Engines**
   - MLX for macOS (Apple Silicon optimization)
   - Planned: ONNX for Windows
   - Planned: Android NN API integration
   - Planned: iOS Core ML integration

### Example Scenario:

```
User Request: "Summarize this document"
├─ Edge Router detects: MacBook Pro M2, 16GB RAM
├─ Decision: Can run Llama 3.2 3B locally via MLX
└─ Routes to: Local MLX engine

User Request: "Analyze this complex legal document"
├─ Edge Router detects: Same device
├─ Decision: Requires Llama 3.1 70B (too large for local)
└─ Routes to: Cloud provider (via LiteLLM)
```

### Division of Responsibilities:

Rather than competing with existing tools, our edge router complements them:

```
Client Request → Edge Router (hardware-aware)
                 ↓                    ↓
            Local Execution      Cloud Routing
            (MLX/ONNX/etc.)     (LiteLLM/OpenRouter)
                 ↓                    ↓
            Local Models        Remote Providers
                                (OpenAI/Anthropic/etc.)
```


- **Edge Router**: Handles "run local or remote?" decision
- **LiteLLM/OpenRouter**: Handles "which remote provider?" decision

### Integration Strategy:

1. **Use LiteLLM for remote routing** - When edge router determines a request needs cloud processing, delegate to LiteLLM for multi-provider routing
2. **Interoperate via OpenAI-compatible API** - Edge router can sit in front of LiteLLM, Ollama, or other tools
3. **Focus on differentiator** - Hardware capability detection and edge/cloud decision-making

## Conclusion

Our edge router addresses a gap in the current LLM infrastructure landscape:

- **Existing tools excel at cloud-to-cloud routing** but ignore edge capabilities
- **Local-first tools run models locally** but lack intelligent routing
- **Our edge router bridges this gap** with hardware-aware, automatic local/remote routing

This edge-aware routing with automatic local/remote fallback based on device capabilities represents a novel contribution to the LLM infrastructure space. Most existing solutions require manual configuration to specify which models run where, rather than automatically detecting "this device can run Llama 3.2 3B locally, but needs to route Llama 3.1 70B to the cloud."

## References

- [LiteLLM Documentation](https://docs.litellm.ai/)
- [OpenRouter](https://openrouter.ai/)
- [RouteLLM Paper](https://arxiv.org/abs/2406.18665)
- [Portkey](https://portkey.ai/)
- [BentoML](https://www.bentoml.com/)