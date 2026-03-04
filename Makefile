# PSFuzz Makefile
# Version 1.0.0

.PHONY: all build clean install test test-race test-coverage run help docker-build docker-run docker-clean

# Variables
BINARY_NAME=psfuzz
VERSION=1.0.0
BUILD_DIR=build
MAIN_FILE=main.go

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOINSTALL=$(GOCMD) install
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"
RACE_FLAGS=-race

# Default target
all: clean build

# Build the project
build:
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_FILE)
	@echo "✓ Build successful: ./$(BINARY_NAME)"

# Build with race detector (for development)
build-race:
	@echo "Building $(BINARY_NAME) with race detector..."
	$(GOBUILD) $(RACE_FLAGS) -o $(BINARY_NAME) $(MAIN_FILE)
	@echo "✓ Build with race detector successful"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILE)
	GOOS=linux GOARCH=386 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-386 $(MAIN_FILE)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_FILE)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILE)
	GOOS=windows GOARCH=386 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-386.exe $(MAIN_FILE)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILE)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_FILE)
	@echo "✓ Multi-platform build complete in $(BUILD_DIR)/"

# Install the binary
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOINSTALL)
	@echo "✓ Installed successfully"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f default_payload.txt random_payload.txt output.txt test_*.txt
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "✓ Tests complete"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) -v -race ./...
	@echo "✓ Race tests complete"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"
	@echo "Opening coverage report..."
	@open coverage.html 2>/dev/null || xdg-open coverage.html 2>/dev/null || echo "Open coverage.html manually"

# Run parameter test script (flags/modules; requires network, run from project root)
test-params:
	@./scripts/test_all_params.sh

# Run the application (with example URL)
run:
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) -u https://example.com -d default -c 2

# Run with race detector
run-race: build-race
	@echo "Running $(BINARY_NAME) with race detector..."
	./$(BINARY_NAME) -u https://example.com -d default -c 5

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@echo "✓ Format complete"

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
		echo "✓ Lint complete"; \
	else \
		echo "⚠ golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Update dependencies
deps:
	@echo "Updating dependencies..."
	$(GOMOD) tidy
	$(GOMOD) download
	@echo "✓ Dependencies updated"

# Show version
version:
	@echo "PSFuzz version $(VERSION)"

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .
	@echo "✓ Docker image built: $(BINARY_NAME):$(VERSION)"

docker-run:
	@echo "Running Docker container..."
	docker run --rm $(BINARY_NAME):latest -u https://example.com -d default -c 2

docker-clean:
	@echo "Cleaning Docker artifacts..."
	docker rmi $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest 2>/dev/null || true
	@echo "✓ Docker cleanup complete"

# Show help
help:
	@echo "PSFuzz Makefile - Version $(VERSION)"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build Targets:"
	@echo "  all           - Clean and build (default)"
	@echo "  build         - Build the binary"
	@echo "  build-race    - Build with race detector"
	@echo "  build-all     - Build for all platforms"
	@echo "  install       - Install the binary"
	@echo "  clean         - Remove build artifacts"
	@echo ""
	@echo "Test Targets:"
	@echo "  test          - Run tests"
	@echo "  test-race     - Run tests with race detector"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-params   - Run parameter test script (scripts/test_all_params.sh)"
	@echo ""
	@echo "Run Targets:"
	@echo "  run           - Build and run with example"
	@echo "  run-race      - Run with race detector"
	@echo ""
	@echo "Docker Targets:"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  docker-clean  - Remove Docker images"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           - Format Go code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  deps          - Update dependencies"
	@echo ""
	@echo "Other:"
	@echo "  version       - Show version"
	@echo "  help          - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test-coverage"
	@echo "  make docker-build"
	@echo "  make clean && make build"

