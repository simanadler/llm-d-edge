# llm-d Edge Router Makefile

# Variables
BINARY_NAME=edge-router
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w"

# Platforms
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

# Directories
BUILD_DIR=build
CMD_DIR=cmd/edge-router

.PHONY: all build test clean install run fmt lint help

# Default target
all: test build

## help: Display this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
## build-stub: Build the llm-d stub server
build-stub:
	@echo "Building llm-d-stub..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/llm-d-stub ./cmd/llm-d-stub
	@echo "Build complete: $(BUILD_DIR)/llm-d-stub"

## build-with-stub: Build both edge-router and stub server
build-with-stub: build build-stub
	@echo "Both binaries built successfully"

	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## build-all: Build binaries for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		$(GO) build $(GOFLAGS) $(LDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/} \
		./$(CMD_DIR); \
		echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}"; \
	done
	@echo "All builds complete"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-router: Run router tests only
test-router:
	@echo "Running router tests..."
	$(GO) test -v -race ./pkg/router/...

## test-macos: Run macOS engine tests only (macOS only)
test-macos:
	@echo "Running macOS engine tests..."
	$(GO) test -v -race ./internal/macos/...

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

## install: Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) ./$(CMD_DIR)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

## run: Build and run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) --config config.yaml --port 8080

## run-dev: Run with development settings
run-dev: build
	@echo "Running $(BINARY_NAME) in development mode..."

## run-stub: Run the llm-d stub server
run-stub: build-stub
	@echo "Running llm-d stub server..."
	./$(BUILD_DIR)/llm-d-stub --port 8081 --latency-min 100 --latency-max 500

## run-stub-with-auth: Run stub server with API key authentication
run-stub-with-auth: build-stub
	@echo "Running llm-d stub server with authentication..."
	./$(BUILD_DIR)/llm-d-stub --port 8081 --require-api-key --api-key test-api-key

## run-stub-with-errors: Run stub server with simulated errors
run-stub-with-errors: build-stub
	@echo "Running llm-d stub server with 10% error rate..."
	./$(BUILD_DIR)/llm-d-stub --port 8081 --error-rate 0.1

## run-integration: Run edge-router with stub configuration
run-integration: build
	@echo "Running edge-router with stub configuration..."
	./$(BUILD_DIR)/$(BINARY_NAME) --config config.stub.yaml --port 8080

## test-integration: Run integration tests with stub server
test-integration: build-with-stub
	@echo "Starting integration tests..."
	@echo "1. Starting stub server in background..."
	@./$(BUILD_DIR)/llm-d-stub --port 8081 --log-requests=false > /dev/null 2>&1 & \
	STUB_PID=$$!; \
	echo "   Stub server PID: $$STUB_PID"; \
	sleep 2; \
	echo "2. Testing stub server health..."; \
	curl -s http://localhost:8081/health > /dev/null && echo "   ✓ Stub server is healthy" || (echo "   ✗ Stub server failed"; kill $$STUB_PID; exit 1); \
	echo "3. Starting edge-router in background..."; \
	./$(BUILD_DIR)/$(BINARY_NAME) --config config.stub.yaml --port 8080 > /dev/null 2>&1 & \
	ROUTER_PID=$$!; \
	echo "   Edge router PID: $$ROUTER_PID"; \
	sleep 2; \
	echo "4. Testing edge-router health..."; \
	curl -s http://localhost:8080/health > /dev/null && echo "   ✓ Edge router is healthy" || (echo "   ✗ Edge router failed"; kill $$STUB_PID $$ROUTER_PID; exit 1); \
	echo "5. Testing remote routing (via stub)..."; \
	curl -s -X POST http://localhost:8080/v1/chat/completions \
		-H "Content-Type: application/json" \
		-d '{"model":"test-model","messages":[{"role":"user","content":"Hello from integration test"}]}' \
		| grep -q "llm-d-stub" && echo "   ✓ Remote routing works" || echo "   ✗ Remote routing failed"; \
	echo "6. Cleaning up..."; \
	kill $$STUB_PID $$ROUTER_PID 2>/dev/null || true; \
	echo "Integration tests complete!"

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Formatting complete"

## lint: Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## mod-tidy: Tidy go modules
mod-tidy:
	@echo "Tidying modules..."
	$(GO) mod tidy
	@echo "Modules tidied"

## mod-download: Download dependencies
mod-download:
	@echo "Downloading dependencies..."
	$(GO) mod download
	@echo "Dependencies downloaded"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t llm-d/edge-router:latest .
	@echo "Docker image built"

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 -v $(PWD)/config.yaml:/app/config.yaml llm-d/edge-router:latest

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed"

## deps: Install development dependencies
deps:
	@echo "Installing development dependencies..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Dependencies installed"

## version: Display version information
version:
	@echo "llm-d Edge Router"
	@echo "Go version: $$($(GO) version)"
	@echo "Git commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# Platform-specific builds
## build-macos: Build for macOS (both Intel and Apple Silicon)
build-macos:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-macos-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-macos-arm64 ./$(CMD_DIR)
	@echo "macOS builds complete"

## build-windows: Build for Windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe ./$(CMD_DIR)
	@echo "Windows build complete"

## build-linux: Build for Linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	@echo "Linux builds complete"

# Development helpers
## dev-setup: Set up development environment
dev-setup: deps mod-download
	@echo "Development environment ready"
	@echo "Copy config.example.yaml to config.yaml and edit as needed"

## watch: Watch for changes and rebuild (requires entr)
watch:
	@if command -v entr > /dev/null; then \
		find . -name '*.go' | entr -r make run-dev; \
	else \
		echo "entr not installed. Install with: brew install entr (macOS) or apt-get install entr (Linux)"; \
	fi