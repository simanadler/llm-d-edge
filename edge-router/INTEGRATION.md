# llm-d Edge Router Integration Guide

This guide explains how to integrate the llm-d Edge Router with the existing llm-d Kubernetes infrastructure.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Integration Patterns](#integration-patterns)
- [Configuration](#configuration)
- [Deployment Scenarios](#deployment-scenarios)
- [Security Considerations](#security-considerations)
- [Monitoring and Observability](#monitoring-and-observability)
- [Troubleshooting](#troubleshooting)

## Architecture Overview

The Edge Router acts as a bridge between edge devices and the llm-d Kubernetes cluster:

```
┌─────────────────────────────────────────────────────────────┐
│                        Edge Device                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              llm-d Edge Router                       │  │
│  │  ┌────────────┐  ┌──────────────┐  ┌─────────────┐ │  │
│  │  │  Routing   │  │   Platform   │  │   Remote    │ │  │
│  │  │   Logic    │──│   Inference  │  │   Client    │ │  │
│  │  │            │  │   Engine     │  │             │ │  │
│  │  └────────────┘  └──────────────┘  └──────┬──────┘ │  │
│  └────────────────────────────────────────────┼────────┘  │
└─────────────────────────────────────────────────┼──────────┘
                                                  │
                                                  │ HTTPS
                                                  │
┌─────────────────────────────────────────────────┼──────────┐
│                  Kubernetes Cluster             │          │
│  ┌──────────────────────────────────────────────▼────────┐ │
│  │              Gateway (Istio/K-Gateway)               │ │
│  └──────────────────────────────────────────────┬────────┘ │
│                                                  │          │
│  ┌──────────────────────────────────────────────▼────────┐ │
│  │              InferencePool Service                   │ │
│  └──────────────────────────────────────────────┬────────┘ │
│                                                  │          │
│  ┌──────────────────────────────────────────────▼────────┐ │
│  │              Model Server Pods (vLLM)                │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐          │ │
│  │  │  Pod 1   │  │  Pod 2   │  │  Pod N   │          │ │
│  │  └──────────┘  └──────────┘  └──────────┘          │ │
│  └──────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Integration Patterns

### 1. Direct Gateway Integration

Edge Router connects directly to the llm-d Gateway:

**Configuration:**
```yaml
remote:
  endpoint: "https://llm-d-gateway.example.com"
  api_key: "${LLM_D_API_KEY}"
  timeout: 30s
  
routing:
  policy: "hybrid"
  fallback_to_remote: true
```

**Use Case:** Production deployments with centralized gateway management.

### 2. Service Mesh Integration

Edge Router integrates with Istio service mesh:

**Configuration:**
```yaml
remote:
  endpoint: "https://llm-d-gateway.example.com"
  api_key: "${LLM_D_API_KEY}"
  headers:
    X-Mesh-Client: "edge-router"
    X-User-ID: "${USER_ID}"
  
routing:
  policy: "latency-optimized"
```

**Use Case:** Multi-tenant environments with advanced traffic management.

### 3. Multi-Cluster Integration

Edge Router can route to multiple llm-d clusters:

**Configuration:**
```yaml
remote:
  endpoint: "https://primary-cluster.example.com"
  fallback_endpoints:
    - "https://secondary-cluster.example.com"
    - "https://tertiary-cluster.example.com"
  
routing:
  policy: "cost-optimized"
  routing_rules:
    - condition: "latency > 500ms"
      action: "use_fallback"
```

**Use Case:** High availability and disaster recovery scenarios.

## Configuration

### Environment Variables

The Edge Router supports environment variable substitution in configuration:

```yaml
remote:
  endpoint: "${LLM_D_ENDPOINT}"
  api_key: "${LLM_D_API_KEY}"
  
local:
  models_dir: "${HOME}/llm-d/models"
```

**Required Environment Variables:**
- `LLM_D_ENDPOINT`: llm-d Gateway endpoint URL
- `LLM_D_API_KEY`: Authentication token for llm-d cluster

**Optional Environment Variables:**
- `LLM_D_TIMEOUT`: Request timeout (default: 30s)
- `LLM_D_MAX_RETRIES`: Maximum retry attempts (default: 3)
- `USER_ID`: User identifier for multi-tenant setups

### Authentication

#### API Key Authentication

```yaml
remote:
  api_key: "${LLM_D_API_KEY}"
```

Set the environment variable:
```bash
export LLM_D_API_KEY="your-api-key-here"
```

#### OAuth2 / OIDC (Future)

```yaml
remote:
  auth:
    type: "oauth2"
    token_url: "https://auth.example.com/token"
    client_id: "${OAUTH_CLIENT_ID}"
    client_secret: "${OAUTH_CLIENT_SECRET}"
```

#### mTLS (Future)

```yaml
remote:
  tls:
    cert_file: "/path/to/client-cert.pem"
    key_file: "/path/to/client-key.pem"
    ca_file: "/path/to/ca-cert.pem"
```

## Deployment Scenarios

### Scenario 1: Developer Workstation

**Goal:** Local development with fallback to production cluster.

**Setup:**
```bash
# Install Edge Router
brew install llm-d-edge-router  # macOS
# or
go install github.com/llm-d-incubation/llm-d-edge/cmd/edge-router@latest

# Configure
cat > ~/.llm-d/config.yaml <<EOF
remote:
  endpoint: "https://llm-d-dev.example.com"
  api_key: "${LLM_D_DEV_API_KEY}"

local:
  models_dir: "~/.llm-d/models"
  platform: "macos"

routing:
  policy: "local-first"
  fallback_to_remote: true
  
  routing_rules:
    - condition: "model_size > 7B"
      action: "route_remote"
EOF

# Run
edge-router --config ~/.llm-d/config.yaml
```

### Scenario 2: Mobile Application

**Goal:** On-device inference with cloud fallback for complex queries.

**Setup (iOS):**
```yaml
# config.yaml embedded in app bundle
remote:
  endpoint: "https://llm-d-mobile.example.com"
  api_key: "${MOBILE_API_KEY}"

local:
  platform: "ios"
  models_dir: "Documents/models"

routing:
  policy: "mobile-optimized"
  fallback_to_remote: true
  
  routing_rules:
    - condition: "battery_level < 20%"
      action: "route_remote"
    - condition: "network_type == 'cellular'"
      action: "prefer_local"
```

## Security Considerations

### 1. API Key Management

**Best Practices:**
- Store API keys in environment variables or secret management systems
- Rotate keys regularly
- Use different keys for different environments (dev, staging, prod)
- Never commit keys to version control

**Example with Kubernetes Secrets:**
```bash
kubectl create secret generic llm-d-secret \
  --from-literal=api-key='your-api-key-here'
```

### 2. Network Security

**TLS/HTTPS:**
- Always use HTTPS for remote endpoints
- Verify TLS certificates
- Consider mTLS for enhanced security

**Firewall Rules:**
```bash
# Allow outbound HTTPS to llm-d cluster
iptables -A OUTPUT -p tcp --dport 443 -d llm-d-gateway.example.com -j ACCEPT

# Allow inbound on Edge Router port (if needed)
iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

### 3. Model Security

**Model Verification:**
- Verify model checksums before loading
- Use signed models when available
- Restrict model directory permissions

```bash
# Set restrictive permissions
chmod 700 ~/.llm-d/models
chmod 600 ~/.llm-d/models/*.gguf
```

### 4. Rate Limiting

**Configuration:**
```yaml
routing:
  rate_limits:
    local:
      requests_per_minute: 60
      burst: 10
    remote:
      requests_per_minute: 100
      burst: 20
```

## Monitoring and Observability

### Metrics

The Edge Router exposes Prometheus metrics at `/metrics`:

**Key Metrics:**
- `edge_router_requests_total`: Total requests by route (local/remote)
- `edge_router_request_duration_seconds`: Request latency histogram
- `edge_router_errors_total`: Total errors by type
- `edge_router_local_model_loaded`: Local model status (0/1)
- `edge_router_queue_depth`: Current request queue depth

**Prometheus Configuration:**
```yaml
scrape_configs:
  - job_name: 'edge-router'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### Logging

**Log Levels:**
- `debug`: Detailed routing decisions and engine operations
- `info`: Request routing and model loading (default)
- `warn`: Fallback activations and retries
- `error`: Failures and exceptions

**Configuration:**
```bash
edge-router --log-level debug --log-format json
```

**Structured Logging Example:**
```json
{
  "level": "info",
  "time": "2026-03-11T09:42:54Z",
  "msg": "Request routed",
  "route": "local",
  "model": "llama-3.2-3b",
  "latency_ms": 245,
  "tokens": 150
}
```

### Tracing (Future)

OpenTelemetry integration for distributed tracing:

```yaml
observability:
  tracing:
    enabled: true
    endpoint: "http://jaeger:14268/api/traces"
    service_name: "edge-router"
```

## Troubleshooting

### Common Issues

#### 1. Connection Refused to Remote Endpoint

**Symptoms:**
```
ERROR Failed to connect to remote endpoint: connection refused
```

**Solutions:**
- Verify endpoint URL is correct
- Check network connectivity: `curl https://llm-d-gateway.example.com/health`
- Verify firewall rules allow outbound HTTPS
- Check API key is valid

#### 2. Local Model Not Loading

**Symptoms:**
```
ERROR Failed to load local model: model file not found
```

**Solutions:**
- Verify model path: `ls -la ~/.llm-d/models/`
- Check model format is supported (GGUF for MLX)
- Verify sufficient disk space
- Check file permissions

#### 3. High Latency

**Symptoms:**
- Requests taking longer than expected
- Timeouts occurring

**Solutions:**
- Check routing policy: consider switching to `latency-optimized`
- Monitor queue depth: `curl localhost:8080/metrics | grep queue_depth`
- Reduce `max_local_concurrency` if CPU-bound
- Increase `remote.timeout` if network is slow

#### 4. Memory Issues

**Symptoms:**
```
ERROR Out of memory loading model
```

**Solutions:**
- Use smaller quantized models (Q4, Q5)
- Reduce `max_local_concurrency`
- Enable model offloading (if supported)
- Route large models to remote cluster

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
edge-router --config config.yaml --log-level debug
```

### Health Checks

**Check Edge Router Health:**
```bash
curl http://localhost:8080/health
```

**Expected Response:**
```json
{
  "status": "healthy",
  "local_engine": "ready",
  "remote_endpoint": "reachable",
  "uptime_seconds": 3600
}
```

### Support

For additional support:
- GitHub Issues: https://github.com/llm-d-incubation/llm-d-edge/issues
- Documentation: https://llm-d.github.io/llm-d-edge
- Community: https://llm-d.slack.com

## Next Steps

1. Review the [README.md](README.md) for quick start guide
2. Explore [config.example.yaml](config.example.yaml) for configuration options
3. Check platform-specific setup in README.md
4. Join the community for support and updates