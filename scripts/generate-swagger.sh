#!/bin/bash

# Script to generate Swagger JSON from Pamlogix protobuf definitions
# Usage: ./scripts/generate-swagger.sh

set -e

echo "🚀 Pamlogix Swagger JSON Generator"
echo "=================================="

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "❌ Error: protoc is not installed"
    echo "Please install protoc (Protocol Buffers Compiler)"
    echo "Visit: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

echo "✅ protoc found: $(protoc --version)"

# Check if we're in the right directory
if [ ! -f "pamlogix/pamlogix.proto" ]; then
    echo "❌ Error: pamlogix/pamlogix.proto not found"
    echo "Please run this script from the project root directory"
    exit 1
fi

echo "✅ Found pamlogix.proto"

# Update Go modules first
echo "📦 Updating Go modules..."
go mod tidy

# Install required tools
echo "🔧 Installing required protobuf tools..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

echo "✅ Tools installed successfully"

# Add Go bin to PATH for this session
export PATH=$PATH:$(go env GOPATH)/bin

# Create output directory
mkdir -p api

# Download required googleapis proto files if not present
if [ ! -f "third_party/google/api/annotations.proto" ]; then
    echo "📥 Downloading required googleapis proto files..."
    mkdir -p third_party/google/api
    curl -s -L https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto -o third_party/google/api/annotations.proto
    curl -s -L https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto -o third_party/google/api/http.proto
    echo "✅ Downloaded googleapis proto files"
fi

# Generate Swagger JSON
echo "🏗️  Generating Swagger JSON..."

protoc \
    -I=pamlogix \
    -I=third_party \
    -I=vendor/github.com/grpc-ecosystem/grpc-gateway/v2 \
    --openapiv2_out=api \
    --openapiv2_opt=logtostderr=true \
    --openapiv2_opt=use_go_templates=true \
    --openapiv2_opt=allow_merge=true \
    --openapiv2_opt=merge_file_name=pamlogix \
    --openapiv2_opt=openapi_naming_strategy=simple \
    --openapiv2_opt=simple_operation_ids=true \
    pamlogix/pamlogix.proto

if [ -f "api/pamlogix.swagger.json" ]; then
    echo "✅ Swagger JSON generated successfully!"
    echo "📄 Output file: api/pamlogix.swagger.json"
    
    # Show file size and basic info
    SIZE=$(du -h api/pamlogix.swagger.json | cut -f1)
    ENDPOINTS=$(grep -o '"\/v1\/[^"]*"' api/pamlogix.swagger.json | wc -l)
    
    echo "📊 File size: $SIZE"
    echo "🔗 API endpoints: $ENDPOINTS"
    echo ""
    echo "🎉 You can now:"
    echo "   • View the JSON: cat api/pamlogix.swagger.json"
    echo "   • Import into Postman for API testing"
    echo "   • Use with Swagger UI for documentation"
    echo "   • Generate client SDKs using swagger-codegen"
    echo ""
    echo "🌐 To view in Swagger UI, visit:"
    echo "   https://editor.swagger.io/ and paste the JSON content"
else
    echo "❌ Error: Swagger JSON generation failed"
    exit 1
fi 