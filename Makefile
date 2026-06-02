.PHONY: build test lint clean fmt vet release

# Build variables
BINARY_NAME := sfs
BUILD_DIR := dist
GO := go

# Version (override with: make build VERSION=1.2.3)
VERSION ?= 0.2.0

# Supported platforms
PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64

## build: Compile the binary for current platform
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	$(GO) build -ldflags "-s -w -X smallFileSync/model.AppVersion=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

## build-all: Compile release archives for all platforms
build-all: $(PLATFORMS)

$(PLATFORMS):
	@GOOS=$(word 1,$(subst -, ,$@)) GOARCH=$(word 2,$(subst -, ,$@)) \
		EXT="" CGO_ENABLED=0 \
		OUTPUT="$(BUILD_DIR)/$(BINARY_NAME)-$(word 1,$(subst -, ,$@))-$(word 2,$(subst -, ,$@))$${EXT}" && \
		echo "Building $$OUTPUT..." && \
		GOOS=$(word 1,$(subst -, ,$@)) GOARCH=$(word 2,$(subst -, ,$@)) CGO_ENABLED=0 \
			$(GO) build -ldflags "-s -w -X smallFileSync/model.AppVersion=$(VERSION)" -o "$$OUTPUT" . && \
		cd $(BUILD_DIR) && \
		zip "$(BINARY_NAME)-$(word 1,$(subst -, ,$@))-$(word 2,$(subst -, ,$@)).zip" "$(BINARY_NAME)-$(word 1,$(subst -, ,$@))-$(word 2,$(subst -, ,$@))$${EXT}" && \
		sha256sum "$(BINARY_NAME)-$(word 1,$(subst -, ,$@))-$(word 2,$(subst -, ,$@)).zip" | awk '{print $$1}' > "$(BINARY_NAME)-$(word 1,$(subst -, ,$@))-$(word 2,$(subst -, ,$@)).zip.sha256" && \
		echo "Archived: $(BINARY_NAME)-$(word 1,$(subst -, ,$@))-$(word 2,$(subst -, ,$@)).zip"

# Windows special case (handled in build-all via PLATFORMS iteration, EXT set above)

## checksums: Generate SHA256 for existing archives
checksums:
	@cd $(BUILD_DIR) && \
	for zip in *.zip; do \
		sha256sum "$$zip" | awk '{print $$1}' > "$$zip.sha256"; \
		echo "Generated $$zip.sha256"; \
	done

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
	@echo "  build       Compile the binary for current platform"
	@echo "  build-all   Compile release archives for all platforms"
	@echo "  checksums   Generate SHA256 checksums for archives"
	@echo "  test        Run all tests"
	@echo "  lint        Run static analysis (go vet)"
	@echo "  fmt         Format code"
	@echo "  clean       Remove build artifacts"
	@echo "  run         Build and run"
	@echo "  help        Show this help message"
