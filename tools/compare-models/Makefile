.PHONY: build clean test run help

# Build the compare-models binary
build:
	@echo "Building compare-models..."
	@go build -o compare-models

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f compare-models
	@go clean

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Run with example (requires config file)
run:
	@echo "Running compare-models..."
	@./compare-models \
		--messages '[{"role":"user","content":"What is 2+2?"}]' \
		--config ../../edge-router/config.stub.yaml \
		--remote-model "meta-llama/Llama-3.1-70B-Instruct" \
		--verbose

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  build   - Build the compare-models binary"
	@echo "  clean   - Remove build artifacts"
	@echo "  test    - Run tests"
	@echo "  run     - Run with example configuration"
	@echo "  deps    - Install/update dependencies"
	@echo "  help    - Show this help message"

# Made with Bob