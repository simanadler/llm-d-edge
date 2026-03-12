# Integration Testing Guide

This guide explains how to test the llm-d Edge Router with stub servers for integration testing.

## Overview

The integration testing setup includes two stub implementations:
- **llm-d-stub**: A mock server that simulates the remote llm-d Kubernetes cluster (runs as separate process on port 8081)
- **local-stub engine**: A mock inference engine that simulates local model inference (runs within edge-router process)
- **edge-router**: The edge router configured to use both stub implementations
- **config.stub.yaml**: Configuration file for integration testing

### Stub Components

#### llm-d-stub (Remote Cluster Simulator)
- **Purpose**: Simulates the remote Kubernetes cluster with llm-d services
- **Location**: `cmd/llm-d-stub/main.go`
- **Binary**: `build/llm-d-stub`
- **Port**: 8081 (default)
- **Responses**: Include `"Source: llm-d-stub"` to identify remote routing

#### local-stub Engine (Local Inference Simulator)
- **Purpose**: Simulates local model inference without requiring actual models or hardware
- **Location**: `internal/stub/stub_engine.go`
- **Integration**: Runs within the edge-router process
- **Platform**: Registered as `"local-stub"` engine
- **Responses**: Include `"Source: local-stub-engine"` to identify local routing

## Quick Start

### 1. Build Components

```bash
make build-with-stub
```

This builds:
- `build/edge-router` - The edge router binary (includes local-stub engine)
- `build/llm-d-stub` - The remote cluster stub server binary

**Note:** The local-stub engine is built into the edge-router binary and doesn't require a separate build step. It's automatically available when you set `platform: "local-stub"` in the configuration.

### 2. Run Automated Integration Tests

```bash
make test-integration
```

This will:
1. Start the llm-d-stub server on port 8081 (simulates remote cluster)
2. Start the edge router on port 8080 (includes local-stub engine)
3. Run health checks
4. Test remote routing through llm-d-stub
5. Clean up processes

## Manual Testing

### Start the Remote Cluster Stub Server (llm-d-stub)

**Basic llm-d-stub server:**
```bash
make run-stub
# Or manually:
./build/llm-d-stub --port 8081
```

**With authentication:**
```bash
make run-stub-with-auth
# Or manually:
./build/llm-d-stub --port 8081 --require-api-key --api-key test-api-key
```

**With simulated errors:**
```bash
make run-stub-with-errors
# Or manually:
./build/llm-d-stub --port 8081 --error-rate 0.1
```

**Custom configuration:**
```bash
./build/llm-d-stub \
  --port 8081 \
  --latency-min 50 \
  --latency-max 200 \
  --error-rate 0.05 \
  --require-api-key \
  --api-key my-secret-key \
  --log-requests
```

### Start the Edge Router (with local-stub engine)

In a separate terminal:

```bash
make run-integration
# Or manually:
./build/edge-router --config config.stub.yaml --port 8080
```

**Note:** The edge-router binary automatically includes the local-stub engine. When you set `platform: "local-stub"` in `config.stub.yaml`, the edge router will use the local-stub engine for local inference simulation.

## Testing Scenarios

### 1. Test Stub Server Directly

**Health check:**
```bash
curl http://localhost:8081/health
```

**Info endpoint:**
```bash
curl http://localhost:8081/
```

**Chat completion:**
```bash
curl -X POST http://localhost:8081/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test-model",
    "messages": [
      {"role": "user", "content": "Hello from stub test"}
    ]
  }'
```

**Expected response:**
```json
{
  "id": "chatcmpl-stub-...",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "test-model",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "... (Model: test-model, Source: llm-d-stub)"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 5,
    "completion_tokens": 20,
    "total_tokens": 25
  }
}
```

### 2. Test Edge Router with Stub

**Health check:**
```bash
curl http://localhost:8080/health
```

**Chat completion (routed to stub):**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test-model",
    "messages": [
      {"role": "user", "content": "Test remote routing"}
    ]
  }'
```

The response should contain `"Source: llm-d-stub"` indicating it was routed to the remote stub server.

**Chat completion (routed to local-stub):**
```bash
# Configure for local-first routing in config.stub.yaml first
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local-stub-model-small",
    "messages": [
      {"role": "user", "content": "Test local routing"}
    ]
  }'
```

The response should contain `"Source: local-stub-engine"` indicating it was routed to the local-stub engine.

### 3. Test Different Routing Policies

Edit `config.stub.yaml` and change the routing policy:

**Remote-first (always use stub):**
```yaml
routing:
  policy: "remote-first"
```

**Local-first (try local, fallback to stub):**
```yaml
routing:
  policy: "local-first"
```

**Hybrid (intelligent routing):**
```yaml
routing:
  policy: "hybrid"
```

Restart the edge router after changing configuration.

### 4. Test Routing Rules

The stub configuration includes routing rules:

**Large model request (routes to stub):**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama-70b",
    "messages": [
      {"role": "user", "content": "This should route to remote"}
    ]
  }'
```

**Long prompt (routes to stub):**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test-model",
    "messages": [
      {"role": "user", "content": "'"$(python3 -c 'print("word " * 1000)')"'"}
    ]
  }'
```

### 5. Test Fallback Behavior

**Stop the stub server** while edge router is running:
```bash
# Kill stub server
pkill llm-d-stub
```

**Send request:**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test-model",
    "messages": [
      {"role": "user", "content": "Test fallback"}
    ]
  }'
```

With `fallback_to_local: true`, the edge router should attempt local inference using the local-stub engine.

### 6. Test Local-Stub Engine

The local-stub engine runs within the edge-router process and simulates local inference.

**Test local-stub directly (with local-first policy):**
```bash
# First, update config.stub.yaml to use local-first policy
# Then restart edge-router and test:
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local-stub-model-small",
    "messages": [
      {"role": "user", "content": "Test local-stub engine"}
    ]
  }'
```

**Expected response:**
```json
{
  "id": "local-stub-...",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "local-stub-model-small",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "... (Model: local-stub-model-small, Source: local-stub-engine)"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 5,
    "completion_tokens": 20,
    "total_tokens": 25
  }
}
```

**Verify routing to local-stub:**
The response should contain `"Source: local-stub-engine"` to confirm it was handled by the local-stub engine, not routed to the remote llm-d-stub server.

### 7. Test Error Handling

**Start stub with errors:**
```bash
./build/llm-d-stub --port 8081 --error-rate 0.5
```

**Send multiple requests:**
```bash
for i in {1..10}; do
  echo "Request $i:"
  curl -X POST http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d '{"model":"test","messages":[{"role":"user","content":"Test"}]}' \
    2>/dev/null | jq -r '.choices[0].message.content // .error.message'
  echo ""
done
```

You should see a mix of successful responses and errors.

### 8. Test Authentication

**Start stub with auth:**
```bash
./build/llm-d-stub --port 8081 --require-api-key --api-key test-api-key
```

**Update config.stub.yaml:**
```yaml
remote:
  endpoint: "http://localhost:8081"
  api_key: "test-api-key"  # or use env var
```

**Test with valid key:**
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"test","messages":[{"role":"user","content":"Auth test"}]}'
```

Should succeed.

**Test with invalid key (direct to stub):**
```bash
curl -X POST http://localhost:8081/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer wrong-key" \
  -d '{"model":"test","messages":[{"role":"user","content":"Auth test"}]}'
```

Should return 401 Unauthorized.

## Monitoring

### Check Metrics

**Stub server metrics:**
```bash
curl http://localhost:8081/metrics
```

**Edge router metrics:**
```bash
curl http://localhost:8080/metrics
```

### View Logs

**Stub server logs:**
The stub server logs all requests when `--log-requests` is enabled (default).

**Edge router logs:**
Set log level in config or via CLI:
```bash
./build/edge-router --config config.stub.yaml --log-level debug
```

## Troubleshooting

### Stub Server Won't Start

**Check if port is in use:**
```bash
lsof -i :8081
```

**Use different port:**
```bash
./build/llm-d-stub --port 8082
# Update config.stub.yaml accordingly
```

### Edge Router Can't Connect to Stub

**Verify stub is running:**
```bash
curl http://localhost:8081/health
```

**Check configuration:**
```bash
cat config.stub.yaml | grep endpoint
```

**Check logs:**
```bash
./build/edge-router --config config.stub.yaml --log-level debug
```

### Requests Timing Out

**Increase timeout in config.stub.yaml:**
```yaml
remote:
  timeout: 60s
```

**Check stub latency settings:**
```bash
./build/llm-d-stub --latency-min 100 --latency-max 500
```

### Authentication Failures

**Verify API key matches:**
- Stub server: `--api-key test-api-key`
- Config: `api_key: "test-api-key"`

**Check environment variable:**
```bash
echo $STUB_API_KEY
```

## Advanced Testing

### Load Testing

Use a tool like `hey` or `ab`:

```bash
# Install hey
go install github.com/rakyll/hey@latest

# Run load test
hey -n 100 -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"model":"test","messages":[{"role":"user","content":"Load test"}]}' \
  http://localhost:8080/v1/chat/completions
```

### Latency Testing

**Test different latency ranges:**
```bash
# Low latency
./build/llm-d-stub --latency-min 10 --latency-max 50

# High latency
./build/llm-d-stub --latency-min 500 --latency-max 2000
```

**Measure end-to-end latency:**
```bash
time curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"test","messages":[{"role":"user","content":"Latency test"}]}'
```

### Concurrent Requests

```bash
# Send 10 concurrent requests
for i in {1..10}; do
  curl -X POST http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d '{"model":"test","messages":[{"role":"user","content":"Concurrent '$i'"}]}' &
done
wait
```

## Cleanup

**Stop all processes:**
```bash
pkill edge-router
pkill llm-d-stub
```

**Or use the integration test cleanup:**
```bash
# The test-integration target automatically cleans up
make test-integration
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Build
        run: cd edge-router && make build-with-stub
      
      - name: Run Integration Tests
        run: cd edge-router && make test-integration
```

## Next Steps

1. **Add more test scenarios** to `config.stub.yaml`
2. **Create automated test scripts** for different routing policies
3. **Integrate with CI/CD pipeline**
4. **Add performance benchmarks**
5. **Test with real llm-d cluster** once available

## Support

For issues or questions:
- Check logs with `--log-level debug`
- Review configuration in `config.stub.yaml`
- See main [README.md](README.md) for general documentation
- See [INTEGRATION.md](INTEGRATION.md) for production integration