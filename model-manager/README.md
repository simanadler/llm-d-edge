# Model Manager

Advanced Model Manager for llm-d-edge - Intelligently recommends and manages local LLM models based on device capabilities and user needs.

## Overview

The Model Manager is a cross-platform tool that:

- **Profiles device hardware** - Detects CPU, memory, GPU, and storage capabilities
- **Recommends optimal models** - Suggests models that will run well on your device
- **Scores and ranks models** - Uses multi-criteria scoring to rank recommendations
- **Provides explanations** - Explains why each model is recommended
- **Cross-platform support** - Works on macOS (with Windows, Linux, iOS, Android planned)

## Features

### Current (v0.2.0 - macOS)

- ✅ Device profiling for macOS (Apple Silicon and Intel)
- ✅ Hardware detection (CPU, Memory, GPU, Storage)
- ✅ Model compatibility analysis
- ✅ Multi-criteria scoring engine
- ✅ Intelligent model recommendations
- ✅ Model download from HuggingFace
- ✅ MLX model conversion (macOS)
- ✅ Model installation and management
- ✅ Edge-router configuration updates
- ✅ Interactive model selection
- ✅ CLI interface with comprehensive commands
- ✅ JSON output support

### Planned

- ⏳ Windows support
- ⏳ Linux support
- ⏳ iOS support
- ⏳ Android support
- ⏳ GGUF conversion support
- ⏳ Usage-based learning
- ⏳ Performance monitoring

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/llm-d-incubation/llm-d-edge.git
cd llm-d-edge/model-manager

# Install dependencies
make deps

# Build
make build

# Install (optional)
make install
```

### Using Go

```bash
go install github.com/llm-d-incubation/llm-d-edge/model-manager/cmd/model-manager@latest
```

## Usage

### 1. Profile Your Device

Get detailed information about your device's capabilities:

```bash
model-manager profile
```

Example output:
```json
{
  "platform": "darwin",
  "cpu": {
    "architecture": "arm64",
    "model": "Apple M2",
    "cores": 8,
    "threads_per_core": 1,
    "is_apple_silicon": true
  },
  "memory": {
    "total_gb": 16.0,
    "available_gb": 11.2,
    "unified": true,
    "type": "LPDDR5"
  },
  "capabilities": {
    "model_size_ranges": {
      "0.5B-1B": {
        "feasibility": 1.0,
        "performance": "excellent",
        "estimated_tps": 75
      },
      "3B-7B": {
        "feasibility": 0.9,
        "performance": "good",
        "estimated_tps": 45
      }
    }
  }
}
```

### 2. Get Model Recommendations

Get personalized model recommendations based on your device and tasks:

```bash
# Basic recommendations
model-manager recommend

# Recommendations for specific tasks
model-manager recommend --tasks code,chat

# High quality preference
model-manager recommend --tasks code --quality high

# Interactive use (fast response needed)
model-manager recommend --quality high --responsiveness interactive

# Batch processing (quality over speed)
model-manager recommend --quality high --responsiveness batch

# All options
model-manager recommend --tasks code,reasoning --quality high --responsiveness balanced
```

**Available Options:**
- `--tasks, -t`: Primary tasks (comma-separated): `general`, `chat`, `code`, `reasoning`, `writing`, `creative`
- `--quality, -q`: Quality preference: `low`, `medium`, `high`, `premium` (default: `medium`)
- `--responsiveness, -r`: Responsiveness level: `interactive`, `balanced`, `batch` (user-friendly alternative to latency)
  - `interactive`: Fast response needed (chat, coding) - prioritizes speed
  - `balanced`: Normal use - balanced speed and quality
  - `batch`: Background processing - prioritizes quality over speed
- `--privacy, -p`: Privacy requirement: `local_only`, `local_preferred`, `cloud_acceptable` (default: `local_preferred`)
- `--storage, -s`: Storage limit in GB (default: 100)
- `--latency, -l`: Latency tolerance in milliseconds (advanced, use `--responsiveness` instead)

#### How Preferences Affect Recommendations

The `--quality` and `--responsiveness` flags significantly impact which models are recommended:

**Quality Preference Examples:**

```bash
# High quality - recommends larger, more capable models
$ model-manager recommend --quality high
Rank 1: Qwen2.5-14B-Instruct (14B)  # Larger model prioritized
Rank 2: Qwen2.5-32B-Instruct (32B)
Rank 3: Llama-3.1-8B-Instruct (8B)

# Low quality - recommends smaller, efficient models
$ model-manager recommend --quality low
Rank 1: Qwen2.5-3B-Instruct (3B)    # Smaller model prioritized
Rank 2: Llama-3.2-3B-Instruct (3B)
Rank 3: SmolLM-1.7B-Instruct (1.7B)
```

**Responsiveness Examples:**

```bash
# Interactive - prioritizes faster models
$ model-manager recommend --quality high --responsiveness interactive
Rank 1: Qwen2.5-14B-Instruct (14B)  # Faster 14B model first
Rank 2: Qwen2.5-32B-Instruct (32B)

# Batch - prioritizes quality over speed
$ model-manager recommend --quality high --responsiveness batch
Rank 1: Qwen2.5-32B-Instruct (32B)  # Larger, higher-quality model first
Rank 2: Qwen2.5-14B-Instruct (14B)
```

**Combined Examples:**

```bash
# Interactive chat/coding - fast, high quality
$ model-manager recommend --quality high --responsiveness interactive
# Result: Qwen2.5-14B-Instruct (14B) - fast and capable

# Batch analysis - maximum quality
$ model-manager recommend --quality high --responsiveness batch
# Result: Qwen2.5-32B-Instruct (32B) - best quality available

# Quick tasks - small and fast
$ model-manager recommend --quality low --responsiveness interactive
# Result: Qwen2.5-3B-Instruct (3B) - minimal resources, fast response
```

Example output:
```
Model Recommendations
====================
Quality Preference: high

Rank 1: Qwen2.5-14B-Instruct (14B)
  Score: 0.85
  Summary: Score: 4.56 - Excellent fit for your device hardware
  Format: model-manager install Qwen2.5-14B-Instruct --format mlx --quantization q8
  Quantization: 18 minutes
  Strengths:
    - Excellent fit for your device hardware
    - Well-suited for your typical tasks
    - Highly efficient (good performance for size)
    - Fast inference (~160 tokens/sec)
```

### 3. Interactive Model Selection and Installation

Select and install models interactively:

```bash
# Interactive selection with recommendations
model-manager select --tasks code,chat

# High quality for batch processing
model-manager select --quality high --responsiveness batch

# Fast models for interactive use
model-manager select --quality high --responsiveness interactive

# With specific format and quantization
model-manager select --tasks code --format mlx --quantization q8

# Specify YAML output location
model-manager select --tasks code --yaml-output /path/to/installed-models.yaml
```

**What it does:**
1. Profiles your device
2. Shows personalized recommendations (influenced by quality and responsiveness)
3. Lets you select one or more models
4. Downloads and converts models (MLX on macOS)
5. Installs to platform-specific directory
6. Generates YAML configuration file with model capabilities

**Available Options:**
- `--tasks, -t`: Primary tasks for recommendations
- `--quality, -q`: Quality preference: `low`, `medium`, `high`, `premium`
- `--responsiveness, -r`: Responsiveness level: `interactive`, `balanced`, `batch`
- `--format, -f`: Model format: `mlx`, `gguf`, `safetensors` (auto-detected if not specified)
- `--quantization`: Quantization level: `q4`, `q8`, `fp16` (auto-selected if not specified)
- `--yaml-output`: Path for generated YAML file (default: models-dir/../installed-models.yaml)
- `--models-dir`: Directory to install models (default: platform-specific)

### 4. Direct Model Installation

Install a specific model directly:

```bash
# Install with auto-detection
model-manager install Qwen/Qwen2.5-3B-Instruct

# Specify format and quantization
model-manager install Qwen/Qwen2.5-3B-Instruct --format mlx --quantization q8

# Specify YAML output location
model-manager install Qwen/Qwen2.5-3B-Instruct --yaml-output /path/to/installed-models.yaml
```

**What it does:**
1. Downloads model from HuggingFace
2. Converts to optimal format (MLX on macOS)
3. Installs to platform-specific directory
4. Generates YAML configuration file with model capabilities

**Available Options:**
- `--format, -f`: Model format (auto-detected if not specified)
- `--quantization`: Quantization level (auto-selected based on device if not specified)
- `--yaml-output`: Path for generated YAML file (default: models-dir/../installed-models.yaml)
- `--models-dir`: Directory to install models

### 5. List Installed Models

View all installed models:

```bash
model-manager list
```

Shows both:
- Models installed by model-manager (with full metadata)
- Manually installed models (detected by file patterns)

### 6. Generate YAML Configuration

Generate or regenerate the YAML configuration file for all installed models:

```bash
# Generate with default location
model-manager generate-yaml

# Specify output location
model-manager generate-yaml --yaml-output /path/to/installed-models.yaml
```

This is useful if:
- The YAML file was deleted or corrupted
- You want to regenerate with updated model metadata
- You manually installed models and want to include them

### 7. Uninstall Models

Remove an installed model:

```bash
model-manager uninstall model-name
```

This removes:
- Model files from disk
- Updates the YAML configuration file to reflect remaining models

### 8. JSON Output

For programmatic use:

```bash
model-manager recommend-json
model-manager recommend-json --tasks code,chat --quality high
```

## Generated YAML Configuration

When you install or uninstall models, the model-manager automatically generates a YAML configuration file at:
- Default: `~/.local/share/llm-d/installed-models.yaml` (or platform equivalent)
- Custom: Specified with `--yaml-output` flag

This file contains:
- All installed models with their metadata
- Model capabilities (parameter count, context length, quality tier, etc.)
- Task suitability scores (chat, code, reasoning, etc.)
- Domain expertise scores (general, technical, medical, etc.)
- Matching rules (which models can substitute for which patterns)

**Structure matches edge-router/config.with-model-matching.yaml:**

```yaml
edge:
  models:
    local:
      - name: "Qwen/Qwen2.5-3B-Instruct"
        format: "mlx"
        quantization: "4bit"
        priority: 1
        path: "/path/to/model"
        
        capabilities:
          parameter_count: "3B"
          context_length: 8192
          model_family: "qwen"
          quality_tier: "medium"
          
          tasks:
            chat: 0.9
            code: 0.7
            reasoning: 0.6
          
          domains:
            general: 0.9
            technical: 0.7
        
        matching:
          can_substitute:
            - pattern: "*3b*"
            - pattern: "qwen*"
          
          exclude_patterns:
            - "gpt-4*"
            - "*7b*"
```

**Usage:**
1. Review the generated file
2. Copy the models you want into your `edge-router/config.yaml`
3. Customize capabilities and matching rules as needed
4. The edge-router will use this configuration for model selection and routing

## Model Storage Locations

Models are installed in platform-specific directories:

- **macOS**: `~/Library/Application Support/llm-d/models/`
- **Windows**: `%APPDATA%\llm-d\models\`
- **Linux**: `~/.local/share/llm-d/models/`


## Model Conversion (macOS)

On macOS, the model-manager automatically converts HuggingFace models to MLX format using `mlx_lm`:

**Prerequisites:**
```bash
pip install mlx-lm
```

**What happens during conversion:**
1. Downloads model from HuggingFace
2. Converts to MLX format with quantization
3. Automatically selects optimal quantization:
   - **Q8** (8-bit) if device has ample memory (2x model size)
   - **Q4** (4-bit) otherwise
4. Stores in platform-specific directory

**Manual quantization:**
```bash
model-manager install model-name --quantization q4  # Force 4-bit
model-manager install model-name --quantization q8  # Force 8-bit
```

## Architecture

### Components

```
model-manager/
├── cmd/model-manager/     # CLI application
├── pkg/
│   ├── types/            # Core data types
│   ├── platform/         # Platform abstraction
│   ├── recommender/      # Recommendation engine
│   └── config/           # Configuration (planned)
└── internal/
    └── macos/            # macOS-specific implementation
```

### Key Concepts

#### Device Profiling

The profiler detects:
- CPU architecture and cores
- Total and available memory
- GPU capabilities (if present)
- Storage capacity
- Platform-specific features (e.g., Apple Silicon unified memory)

#### Model Compatibility

For each model, the system calculates:
- Memory requirements
- Expected performance (tokens/sec)
- Recommended format (MLX, GGUF, etc.)
- Recommended quantization (Q4, Q8, FP16)
- Compatibility confidence score

#### Scoring Engine

Models are scored on five criteria:
- **Device Fit (30%)** - How well the model fits the hardware
  - Considers memory usage and performance
  - Adjusts based on responsiveness preference (speed vs quality)
- **Task Alignment (35%)** - How well it matches user tasks
  - Matches model capabilities to user's specified tasks
- **Quality (20%)** - Model quality tier and parameter count
  - Adjusts based on quality preference (low/medium/high/premium)
  - High quality preference boosts larger models
  - Low quality preference boosts smaller, efficient models
- **Efficiency (10%)** - Performance per GB
  - For interactive use: prioritizes speed per GB
  - For batch use: prioritizes quality per GB
- **Accessibility (5%)** - Ease of installation
  - Download size and license considerations

**Adaptive Scoring:**
The scoring weights automatically adjust based on device capabilities:
- High-end devices (32GB+): Prioritize quality over efficiency
- Mid-range devices (16-32GB): Balanced approach
- Low-end devices (<16GB): Prioritize efficiency over quality

## Development

### Prerequisites

- Go 1.22 or later
- macOS (for current version)

### Building

```bash
# Build
make build

# Run tests
make test

# Run with race detector
make dev-build

# Format code
make fmt

# Run all checks
make check
```

### Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

### Project Structure

```
model-manager/
├── cmd/
│   └── model-manager/
│       └── main.go              # CLI entry point
├── pkg/
│   ├── types/
│   │   └── types.go             # Core data structures
│   ├── platform/
│   │   ├── interface.go         # Platform interface
│   │   ├── factory.go           # Platform factory
│   │   └── macos.go             # macOS factory (build tag)
│   └── recommender/
│       ├── engine.go            # Main recommendation engine
│       ├── matcher.go           # Model matching and compatibility
│       └── scoring.go           # Scoring and ranking
├── internal/
│   └── macos/
│       ├── profiler.go          # macOS device profiler
│       └── monitor.go           # macOS system monitor
├── docs/                        # Documentation
├── go.mod                       # Go module definition
├── Makefile                     # Build automation
└── README.md                    # This file
```

## Configuration

Configuration support is planned for future releases. It will include:

- User preferences (tasks, quality, privacy)
- Model registry settings
- Storage limits
- Custom scoring weights

## Integration with llm-d-edge

The Model Manager is designed to integrate with the llm-d-edge router:

1. **Device Profiling** - Provides device capabilities to the router
2. **Model Selection** - Recommends models for local inference
3. **Confidence Scoring** - Helps router decide when to use local vs. remote
4. **Model Management** - Downloads and installs models (planned)

## Contributing

Contributions are welcome! Please see the main llm-d-edge repository for contribution guidelines.

## License

[License information to be added]

## Related Documentation

- [Advanced Model Manager Requirements](../docs/advanced-model-manager-requirements.md)
- [Cross-Platform Architecture](../docs/advanced-model-manager-cross-platform-architecture.md)
- [Differentiation Analysis](../docs/advanced-model-manager-differentiation.md)

## Roadmap

### Phase 1: Foundation (Current)
- ✅ macOS device profiling
- ✅ Model compatibility analysis
- ✅ Scoring and ranking
- ✅ CLI interface

### Phase 2: Model Management (Next)
- ⏳ Model download from HuggingFace
- ⏳ Model installation and conversion
- ⏳ Storage management
- ⏳ Model updates

### Phase 3: Intelligence
- ⏳ Usage tracking
- ⏳ Adaptive recommendations
- ⏳ Performance monitoring
- ⏳ Automatic optimization

### Phase 4: Cross-Platform
- ⏳ Windows support
- ⏳ Linux support
- ⏳ iOS support
- ⏳ Android support

## Support

For issues, questions, or contributions, please visit the [llm-d-edge repository](https://github.com/llm-d-incubation/llm-d-edge).