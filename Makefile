# Variables
GO_VERSION := 1.21
PROJECT_NAME := analog-artifact-simulator
WASM_OUTPUT := dist/main.wasm
WASM_EXEC_JS := dist/wasm_exec.js
CMD_DIR := cmd/wasm
WEB_DIR := web
PKG_DIR := pkg
DIST_DIR := dist

# Go build flags
GOOS := js
GOARCH := wasm
GO_BUILD_FLAGS := -ldflags="-s -w"

# Default target
.PHONY: all
all: clean build

# Build WebAssembly module
.PHONY: build
build: $(WASM_OUTPUT) $(WASM_EXEC_JS)
	@echo "Copying web files to dist..."
	@cp -r $(WEB_DIR)/* $(DIST_DIR)/ 2>/dev/null || true
	@echo "Build completed successfully!"

# Build WASM file
$(WASM_OUTPUT): $(shell find $(PKG_DIR) -name "*.go") $(CMD_DIR)/main.go
	@echo "Building WebAssembly module..."
	@mkdir -p $(DIST_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GO_BUILD_FLAGS) -o $(WASM_OUTPUT) $(CMD_DIR)/main.go

# Copy wasm_exec.js from Go installation
$(WASM_EXEC_JS):
	@echo "Copying wasm_exec.js..."
	@if [ -n "$(GOROOT)" ]; then \
		cp "$(GOROOT)/misc/wasm/wasm_exec.js" $(WASM_EXEC_JS); \
	else \
		echo "Warning: GOROOT not set, trying to find Go installation..."; \
		go env GOROOT | xargs -I {} cp "{}/misc/wasm/wasm_exec.js" $(WASM_EXEC_JS); \
	fi


# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(WASM_OUTPUT) $(WASM_EXEC_JS)

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Clean and build (default)"
	@echo "  build        - Build WebAssembly module"
	@echo "  clean        - Clean build artifacts"
	@echo "  help         - Show this help"