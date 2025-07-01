# Variables
GO_VERSION := 1.21
PROJECT_NAME := analog-artifact-simulator
WASM_OUTPUT := dist/main.wasm
WASM_EXEC_JS := dist/wasm_exec.js
CMD_DIR := cmd/wasm
WEB_DIR := web
PKG_DIR := pkg
DIST_DIR := dist
DOCS_DIR := docs
TECH_MD := $(DOCS_DIR)/technical-implementation.md
TECH_HTML := $(DIST_DIR)/technical-implementation.html

# Go build flags
GOOS := js
GOARCH := wasm
GO_BUILD_FLAGS := -ldflags="-s -w"

# Default target
.PHONY: all
all: clean build

# Build WebAssembly module (default - without technical documentation)
.PHONY: build
build: $(WASM_OUTPUT) $(WASM_EXEC_JS)
	@echo "Copying web files to dist..."
	@cp -r $(WEB_DIR)/* $(DIST_DIR)/ 2>/dev/null || true
	@echo "Build completed successfully!"

# Build WebAssembly module with technical documentation
.PHONY: build-with-docs
build-with-docs: $(WASM_OUTPUT) $(WASM_EXEC_JS) $(TECH_HTML)
	@echo "Copying web files to dist..."
	@cp -r $(WEB_DIR)/* $(DIST_DIR)/ 2>/dev/null || true
	@echo "Processing index.html with technical implementation..."
	@if command -v powershell >/dev/null 2>&1; then \
		powershell -Command "\$$content = Get-Content '$(TECH_HTML)' -Raw; \$$html = Get-Content '$(DIST_DIR)/index.html' -Raw; \$$pattern = '(?s)<div id=\"tech-overview\" class=\"section\">.*?</div>'; \$$replacement = '<div id=\"tech-overview\" class=\"section\">' + \$$content + '</div>'; \$$html = \$$html -replace \$$pattern, \$$replacement; Set-Content '$(DIST_DIR)/index.html' \$$html"; \
	else \
		sed -i '/<div id="tech-overview" class="section">/,/<\/div>/c\<div id="tech-overview" class="section">' '$(DIST_DIR)/index.html' && \
		cat '$(TECH_HTML)' >> '$(DIST_DIR)/index.html' && \
		echo '</div>' >> '$(DIST_DIR)/index.html'; \
	fi
	@rm -f $(TECH_HTML)
	@echo "Build completed successfully with technical documentation!"

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

# Convert markdown to HTML using pandoc
$(TECH_HTML): $(TECH_MD)
	@echo "Converting technical implementation markdown to HTML..."
	@mkdir -p $(DIST_DIR)
	@pandoc $(TECH_MD) -o $(TECH_HTML) --mathjax --no-highlight --shift-heading-level-by=1


# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(DIST_DIR) 2>/dev/null || true

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all            - Clean and build (default)"
	@echo "  build          - Build WebAssembly module"
	@echo "  build-with-docs - Build WebAssembly module with technical documentation"
	@echo "  clean          - Clean build artifacts"
	@echo "  help           - Show this help"