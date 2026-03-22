# compare-models - Model Performance Comparison Tool

A CLI tool for comparing inference performance across multiple LLM models (both local and remote) using the same query. This tool helps you benchmark and compare models to make informed decisions about model selection and deployment.

## Features

- 🔄 **Compare Multiple Models**: Test all local models from your edge-router config plus a remote model
- 📊 **Comprehensive Metrics**: Collect latency, throughput, memory usage, and success rates
- 🎯 **Unified Query**: Use the same inference query across all models for fair comparison
- 💾 **JSON Output**: Export results in structured JSON format for analysis
- 🔌 **Code Reuse**: Leverages 80% of code from edge-router for consistency and reliability

## Installation

```bash
# From the llm-d-edge root directory
cd tools/compare-models
go build -o compare-models
```

## Usage

### Basic Usage

```bash
./compare-models \
  --messages '[{"role":"user","content":"What is the capital of France?"}]' \
  --config ../../edge-router/config.yaml \
  --remote-model "meta-llama/Llama-3.1-70B-Instruct"
```

### With Output File

```bash
./compare-models \
  --messages '[{"role":"user","content":"Explain quantum computing"}]' \
  --config ../../edge-router/config.yaml \
  --remote-model "meta-llama/Llama-3.1-70B-Instruct" \
  --output comparison-results.json
```

### With Verbose Logging

```bash
./compare-models \
  --messages '[{"role":"user","content":"Write a Python function"}]' \
  --config ../../edge-router/config.yaml \
  --remote-model "Qwen/Qwen2.5-72B-Instruct" \
  --verbose
```

## Command-Line Flags

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--messages` | `-m` | Yes | JSON array of messages for the inference query |
| `--config` | `-c` | Yes | Path to edge-router configuration file |
| `--remote-model` | `-r` | Yes | Name of the remote model to compare |
| `--output` | `-o` | No | Output JSON file path (defaults to stdout) |
| `--verbose` | `-v` | No | Enable verbose logging |

## Input Format

### Messages Format

The `--messages` flag accepts a JSON array of message objects compatible with OpenAI's chat completion format:

```json
[
  {
    "role": "system",
    "content": "You are a helpful assistant."
  },
  {
    "role": "user",
    "content": "What is machine learning?"
  }
]
```

**Supported roles**: `system`, `user`, `assistant`

### Configuration File

Uses the same configuration format as edge-router. Example:

```yaml
edge:
  platform: "macos"
  
  models:
    local:
      - name: "meta-llama/Llama-3.1-70B-Instruct"
        path: "~/models/llama-3.1-70b"
        format: "mlx"
        quantization: "8bit"
      
      - name: "Qwen/Qwen3-Coder-30B-A3B-Instruct"
        path: "~/models/qwen3-coder-30b"
        format: "mlx"
        quantization: "6bit"
    
    remote:
      cluster_url: "https://api.example.com"
      auth_token: "${API_TOKEN}"
      headers:
        RITS_API_KEY: "${RITS_API_KEY}"
```

## Output Format

### JSON Structure

```json
{
  "timestamp": "2026-03-22T21:30:00Z",
  "query": {
    "messages": [
      {
        "role": "user",
        "content": "What is the capital of France?"
      }
    ],
    "max_tokens": 100,
    "temperature": 0.7
  },
  "models": [
    {
      "model_name": "meta-llama/Llama-3.1-70B-Instruct",
      "model_type": "local",
      "success": true,
      "metrics": {
        "time_to_first_token_ms": 150,
        "total_latency_ms": 2500,
        "tokens_per_second": 45.2,
        "prompt_tokens": 10,
        "completion_tokens": 100,
        "total_tokens": 110,
        "memory_usage_mb": 8192,
        "finish_reason": "stop"
      },
      "response": "The capital of France is Paris..."
    },
    {
      "model_name": "Qwen/Qwen3-Coder-30B-A3B-Instruct",
      "model_type": "local",
      "success": true,
      "metrics": {
        "time_to_first_token_ms": 120,
        "total_latency_ms": 1800,
        "tokens_per_second": 55.6,
        "prompt_tokens": 10,
        "completion_tokens": 100,
        "total_tokens": 110,
        "memory_usage_mb": 6144,
        "finish_reason": "stop"
      },
      "response": "The capital of France is Paris..."
    },
    {
      "model_name": "meta-llama/Llama-3.1-70B-Instruct",
      "model_type": "remote",
      "success": true,
      "metrics": {
        "time_to_first_token_ms": 300,
        "total_latency_ms": 3200,
        "tokens_per_second": 31.3,
        "prompt_tokens": 10,
        "completion_tokens": 100,
        "total_tokens": 110,
        "finish_reason": "stop"
      },
      "response": "The capital of France is Paris..."
    }
  ],
  "summary": {
    "total_models": 3,
    "successful": 3,
    "failed": 0,
    "average_latency_ms": 2500,
    "fastest_model": "Qwen/Qwen3-Coder-30B-A3B-Instruct",
    "slowest_model": "meta-llama/Llama-3.1-70B-Instruct (remote)",
    "highest_tps_model": "Qwen/Qwen3-Coder-30B-A3B-Instruct"
  }
}
```

### Metrics Explained

| Metric | Description |
|--------|-------------|
| `time_to_first_token_ms` | Time until first token is generated (for streaming) or total latency (for non-streaming) |
| `total_latency_ms` | Total time from request to complete response |
| `tokens_per_second` | Throughput: completion tokens / total latency in seconds |
| `prompt_tokens` | Number of tokens in the input prompt |
| `completion_tokens` | Number of tokens in the generated response |
| `total_tokens` | Sum of prompt and completion tokens |
| `memory_usage_mb` | Memory used by the model (local models only) |
| `finish_reason` | Why generation stopped: "stop", "length", "content_filter" |

## Examples

### Example 1: Simple Question

```bash
./compare-models \
  -m '[{"role":"user","content":"What is 2+2?"}]' \
  -c ../../edge-router/config.yaml \
  -r "meta-llama/Llama-3.1-70B-Instruct"
```

### Example 2: Code Generation

```bash
./compare-models \
  -m '[{"role":"user","content":"Write a Python function to calculate fibonacci numbers"}]' \
  -c ../../edge-router/config.yaml \
  -r "Qwen/Qwen2.5-72B-Instruct" \
  -o code-comparison.json
```

### Example 3: Multi-turn Conversation

```bash
./compare-models \
  -m '[
    {"role":"system","content":"You are a helpful coding assistant"},
    {"role":"user","content":"How do I sort a list in Python?"},
    {"role":"assistant","content":"You can use the sorted() function or .sort() method"},
    {"role":"user","content":"What is the difference?"}
  ]' \
  -c ../../edge-router/config.yaml \
  -r "meta-llama/Llama-3.1-70B-Instruct" \
  -o conversation-comparison.json
```

## Use Cases

### 1. Model Selection
Compare different models to choose the best one for your use case based on latency, throughput, and quality.

### 2. Local vs Remote Performance
Understand the performance trade-offs between running models locally versus using remote inference.

### 3. Quantization Impact
Compare different quantization levels (4-bit, 6-bit, 8-bit) to balance quality and performance.

### 4. Hardware Benchmarking
Test how different models perform on your specific hardware configuration.

### 5. Cost Analysis
Combine performance metrics with cost data to optimize for cost-effectiveness.

## Architecture

The tool maximizes code reuse from the edge-router project:

- **80% Reused**: Configuration loading, inference engines, remote client, type definitions
- **20% New**: Comparison orchestration, result aggregation, CLI interface

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed architecture documentation.

## Error Handling

The tool handles errors gracefully:

- **Model Loading Failures**: Continues with other models, reports error in results
- **Inference Timeouts**: Marks model as failed, includes error message
- **Network Errors**: Retries remote requests up to 3 times
- **Partial Failures**: Generates results for successful models even if some fail

## Limitations

1. **Sequential Execution**: Models are tested one at a time (parallel execution planned for future)
2. **Non-Streaming**: Currently measures total latency; streaming TTFT support planned
3. **Memory Monitoring**: Platform-specific; currently best supported on macOS
4. **Single Query**: Each run tests one query; batch testing planned for future

## Troubleshooting

### Model Not Found
```
Error: model not found at path: ~/models/llama-3.1-70b
```
**Solution**: Verify model path in config file and ensure model is downloaded.

### Remote Connection Failed
```
Error: failed to send request: connection refused
```
**Solution**: Check cluster URL, network connectivity, and authentication credentials.

### Insufficient Memory
```
Error: failed to load model: insufficient memory
```
**Solution**: Close other applications or use a smaller/more quantized model.

### Invalid JSON
```
Error: failed to parse messages: invalid JSON
```
**Solution**: Ensure messages JSON is properly formatted with escaped quotes.

## Development

### Building from Source

```bash
cd tools/compare-models
go build -o compare-models
```

### Running Tests

```bash
go test ./...
```

### Code Structure

```
tools/compare-models/
├── main.go                    # CLI entry point
├── pkg/
│   └── compare/
│       ├── compare.go         # Comparison orchestration
│       ├── results.go         # Result structures
│       └── metrics.go         # Metrics calculation
├── README.md                  # This file
├── PLAN.md                    # Implementation plan
└── ARCHITECTURE.md            # Architecture documentation
```

## Contributing

This tool is part of the llm-d-edge project. Contributions should:

1. Maximize code reuse from edge-router
2. Maintain consistency with edge-router patterns
3. Include tests for new functionality
4. Update documentation

## License

Same as llm-d-edge project.

## Related Tools

- **edge-router**: The main inference routing service
- **model-manager**: Model download and management tool

## Support

For issues and questions:
1. Check [ARCHITECTURE.md](ARCHITECTURE.md) for implementation details
2. Review edge-router documentation for config format
3. Open an issue in the llm-d-edge repository