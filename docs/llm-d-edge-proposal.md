# llm-d-edge Proposal

**Proposal for llm-d Edge Support**

**Date**: March 11, 2026  
**Author**: Sima Nadler 
**Version**: 1.0 

---

## Executive Summary

This is a proposal to extend llm-d such that inferencing may be done on models running on edge devices or on the enterprise's centralized inferencing capabilities.  The idea is to leverage edge devices such as laptops, mobile devices, and in the future IoT devices, enabling enterprises to take advantage of the compute power of their employees, customers, and supplier devices in addition to their data center running llm-d. The solution introduces a **Hybrid Edge-Cloud Architecture** where inference requests coming from edge devices are routed either to local models on the device or to remote cluster resources.

**Key Benefits**:
- Substantial reduction in remote inference costs
- <100ms routing overhead
- Offline operation capability
- Privacy-preserving (local data stays local)
- No changes to existing llm-d cluster
- Consistent developer experience

## Current State Analysis

### llm-d Architecture Overview

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

**What's Missing for Edge Devices**:
1. No support for edge device operating systems
2. No client-side routing logic
3. No edge-optimized model formats (MLX, GGUF)
4. No offline operation mode
5. No single-user deployment model
6. Assumes Kubernetes infrastructure

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

