#!/bin/bash
# Check if protobuf dependencies are installed and ready

set -e

echo "🔍 Checking protobuf dependencies..."
echo ""

# Check for protoc
if command -v protoc &> /dev/null; then
    VERSION=$(protoc --version 2>&1 | head -1)
    echo "✅ protoc found: $VERSION"
else
    echo "❌ protoc not found"
    echo "   Install with: brew install protobuf"
    echo "   Or see: INSTALL_PROTOBUF.md"
    exit 1
fi

# Check for protoc-gen-go
if command -v protoc-gen-go &> /dev/null; then
    echo "✅ protoc-gen-go found"
else
    echo "❌ protoc-gen-go not found"
    echo "   Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    echo "   Make sure \$(go env GOPATH)/bin is in your PATH"
    exit 1
fi

# Check for buf (optional)
if command -v buf &> /dev/null; then
    VERSION=$(buf --version 2>&1 | head -1)
    echo "✅ buf found: $VERSION (optional)"
else
    echo "⚠️  buf not found (optional, for linting)"
fi

echo ""
echo "✅ All required dependencies are installed!"
echo ""
echo "Next step: make proto-generate"
