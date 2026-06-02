.PHONY: build test lint clean fmt vet

# Build variables
BINARY_NAME := sfs
BUILD_DIR := dist
GO := go

# Version (override with: make build VERSION=1.2.3)
VERSION ?= 0.2.0

## build: Compile the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	$(GO) build -ldflags "-s -w -X smallFileSync/model.AppVersion=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test ./... -v

## lint: Run static analysis (go vet)
lint:
	@echo "Running go vet..."
	$(GO) vet ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

## vet: Alias for lint
vet: lint

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	@echo "Done."

## run: Build and run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

## help: Show available targets
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build   Compile the binary"
	@echo "  test    Run all tests"
	@echo "  lint    Run static analysis (go vet)"
	@echo "  fmt     Format code"
	@echo "  clean   Remove build artifacts"
	@echo "  run     Build and run"
	@echo "  help    Show this help message"
