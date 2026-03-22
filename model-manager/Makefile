.PHONY: build test clean install run-profile run-recommend

# Build variables
BINARY_NAME=model-manager
BUILD_DIR=bin
GO=go
GOFLAGS=-v

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/model-manager

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Run profile command
run-profile: build
	@echo "Running device profile..."
	@$(BUILD_DIR)/$(BINARY_NAME) profile

# Run recommend command
run-recommend: build
	@echo "Running model recommendations..."
	@$(BUILD_DIR)/$(BINARY_NAME) recommend

# Run recommend-json command
run-recommend-json: build
	@echo "Running model recommendations (JSON)..."
	@$(BUILD_DIR)/$(BINARY_NAME) recommend-json

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# Run all checks
check: fmt test lint

# Development build (with race detector)
dev-build:
	@echo "Building with race detector..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -race $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/model-manager

# Help
help:
	@echo "Available targets:"
	@echo "  build              - Build the binary"
	@echo "  test               - Run tests"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  clean              - Clean build artifacts"
	@echo "  deps               - Install dependencies"
	@echo "  install            - Install the binary"
	@echo "  run-profile        - Run device profile"
	@echo "  run-recommend      - Run model recommendations"
	@echo "  run-recommend-json - Run model recommendations (JSON)"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code"
	@echo "  check              - Run all checks"
	@echo "  dev-build          - Build with race detector"
	@echo "  help               - Show this help"