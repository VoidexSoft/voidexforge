# Makefile for Pamlogix project

# Variables
PROTO_PATH := pamlogix
PROTO_FILE := $(PROTO_PATH)/pamlogix.proto
OUT_DIR := api
SWAGGER_OUT := $(OUT_DIR)/pamlogix.swagger.json
GO_OUT := $(OUT_DIR)/go
THIRD_PARTY_DIR := third_party

# Default target
.PHONY: all
all: swagger

# Check if protoc is installed
.PHONY: check-protoc
check-protoc:
	@echo "🔍 Checking for protoc..."
	@if ! command -v protoc >/dev/null 2>&1; then \
		echo "❌ Error: protoc is not installed"; \
		echo "Please install protoc (Protocol Buffers Compiler)"; \
		echo "Visit: https://grpc.io/docs/protoc-installation/"; \
		exit 1; \
	fi
	@echo "✅ protoc found: $$(protoc --version)"

# Check if proto file exists
.PHONY: check-proto
check-proto:
	@if [ ! -f "$(PROTO_FILE)" ]; then \
		echo "❌ Error: $(PROTO_FILE) not found"; \
		echo "Please run this from the project root directory"; \
		exit 1; \
	fi
	@echo "✅ Found pamlogix.proto"

# Update Go modules
.PHONY: mod-tidy
mod-tidy:
	@echo "📦 Updating Go modules..."
	@go mod tidy

# Install required tools
.PHONY: install-tools
install-tools:
	@echo "🔧 Installing required protobuf tools..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	@echo "✅ Tools installed successfully"

# Download required googleapis proto files
.PHONY: download-googleapis
download-googleapis:
	@if [ ! -f "$(THIRD_PARTY_DIR)/google/api/annotations.proto" ]; then \
		echo "📥 Downloading required googleapis proto files..."; \
		mkdir -p $(THIRD_PARTY_DIR)/google/api; \
		curl -s -L https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto -o $(THIRD_PARTY_DIR)/google/api/annotations.proto; \
		curl -s -L https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto -o $(THIRD_PARTY_DIR)/google/api/http.proto; \
		echo "✅ Downloaded googleapis proto files"; \
	fi

# Generate Swagger JSON
.PHONY: swagger
swagger: check-protoc check-proto mod-tidy install-tools download-googleapis create-dirs
	@echo "🚀 Pamlogix Swagger JSON Generator"
	@echo "=================================="
	@echo "🏗️  Generating Swagger JSON from $(PROTO_FILE)..."
	@export PATH=$$PATH:$$(go env GOPATH)/bin && \
	protoc \
		-I=$(PROTO_PATH) \
		-I=$(THIRD_PARTY_DIR) \
		-I=vendor/github.com/grpc-ecosystem/grpc-gateway/v2 \
		--openapiv2_out=$(OUT_DIR) \
		--openapiv2_opt=logtostderr=true \
		--openapiv2_opt=use_go_templates=true \
		--openapiv2_opt=allow_merge=true \
		--openapiv2_opt=merge_file_name=pamlogix \
		--openapiv2_opt=openapi_naming_strategy=simple \
		--openapiv2_opt=simple_operation_ids=true \
		$(PROTO_FILE)
	@if [ -f "$(SWAGGER_OUT)" ]; then \
		echo "✅ Swagger JSON generated successfully!"; \
		echo "📄 Output file: $(SWAGGER_OUT)"; \
		SIZE=$$(du -h $(SWAGGER_OUT) | cut -f1); \
		ENDPOINTS=$$(grep -o '"/v1/[^"]*"' $(SWAGGER_OUT) | wc -l); \
		echo "📊 File size: $$SIZE"; \
		echo "🔗 API endpoints: $$ENDPOINTS"; \
		echo ""; \
		echo "🎉 You can now:"; \
		echo "   • View the JSON: cat $(SWAGGER_OUT)"; \
		echo "   • Import into Postman for API testing"; \
		echo "   • Use with Swagger UI for documentation"; \
		echo "   • Generate client SDKs using swagger-codegen"; \
		echo ""; \
		echo "🌐 To view in Swagger UI, visit:"; \
		echo "   https://editor.swagger.io/ and paste the JSON content"; \
	else \
		echo "❌ Error: Swagger JSON generation failed"; \
		exit 1; \
	fi

# Generate Go code (optional)
.PHONY: go-gen
go-gen: check-protoc check-proto install-tools create-dirs
	@echo "Generating Go code from $(PROTO_FILE)..."
	@export PATH=$$PATH:$$(go env GOPATH)/bin && \
	protoc \
		-I$(PROTO_PATH) \
		-I$(shell go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway/v2)/third_party/googleapis \
		-I$(shell go list -m -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway/v2) \
		--go_out=$(GO_OUT) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GO_OUT) \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(GO_OUT) \
		--grpc-gateway_opt=paths=source_relative \
		$(PROTO_FILE)
	@echo "Go code generated in $(GO_OUT)"

# Create output directories
.PHONY: create-dirs
create-dirs:
	@mkdir -p $(OUT_DIR)
	@mkdir -p $(GO_OUT)
	@mkdir -p $(THIRD_PARTY_DIR)

# Clean generated files
.PHONY: clean
clean:
	@echo "Cleaning generated files..."
	@rm -rf $(OUT_DIR)
	@rm -rf $(THIRD_PARTY_DIR)

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all            - Generate Swagger JSON (default)"
	@echo "  swagger        - Generate Swagger JSON documentation"
	@echo "  go-gen         - Generate Go code from protobuf"
	@echo "  install-tools  - Install required protobuf tools"
	@echo "  mod-tidy       - Update Go modules and vendor dependencies"
	@echo "  download-googleapis - Download required googleapis proto files"
	@echo "  check-protoc   - Check if protoc is installed"
	@echo "  check-proto    - Check if proto file exists"
	@echo "  clean          - Remove generated files"
	@echo "  help           - Show this help message" 