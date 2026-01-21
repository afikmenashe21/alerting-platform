#!/bin/bash
# Script to verify that generated protobuf code is up-to-date
# This ensures developers don't commit changes to .proto files without regenerating code

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROTO_DIR="$REPO_ROOT/proto"
PKG_PROTO_DIR="$REPO_ROOT/pkg/proto"

echo "🔍 Verifying generated protobuf code is up-to-date..."
echo ""

# Check if buf is available
if ! command -v buf &> /dev/null; then
    echo "❌ buf not found. Install it first: brew install bufbuild/buf/buf"
    exit 1
fi

# Check if protoc-gen-go is available
if ! command -v protoc-gen-go &> /dev/null; then
    echo "❌ protoc-gen-go not found. Install it first:"
    echo "   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    exit 1
fi

# Create temporary directory for fresh generation
TMP_DIR=$(mktemp -d)
TMP_PKG_DIR="$TMP_DIR/pkg/proto"
trap "rm -rf $TMP_DIR" EXIT

# Create temporary buf.gen.yaml with output to temp directory
TMP_BUF_GEN="$TMP_DIR/buf.gen.yaml"
cat > "$TMP_BUF_GEN" << EOF
version: v2
plugins:
  - local: protoc-gen-go
    out: $TMP_PKG_DIR
    opt:
      - module=github.com/afikmenashe/alerting-platform/pkg/proto
      - paths=import
inputs:
  - directory: .
EOF

echo "1️⃣  Generating fresh protobuf code in temporary directory..."
# Use buf generate with temporary config (same settings as proto-generate target)
cd "$PROTO_DIR"
buf generate --template "$TMP_BUF_GEN" || {
    echo "❌ buf generate failed"
    exit 1
}

# Debug: Check what was generated
if [ ! -d "$TMP_PKG_DIR" ]; then
    echo "❌ Output directory not created: $TMP_PKG_DIR"
    exit 1
fi

echo ""
echo "2️⃣  Comparing with existing generated code..."

# Compare each generated file
DIFF_FOUND=0
FILES_CHECKED=0

for proto_type in alerts common notifications rules; do
    EXISTING_FILE="$PKG_PROTO_DIR/$proto_type/$proto_type.pb.go"
    GENERATED_FILE="$TMP_PKG_DIR/$proto_type/$proto_type.pb.go"
    
    if [ ! -f "$EXISTING_FILE" ]; then
        echo "❌ Missing generated file: $EXISTING_FILE"
        DIFF_FOUND=1
        continue
    fi
    
    if [ ! -f "$GENERATED_FILE" ]; then
        echo "❌ Failed to generate: $GENERATED_FILE"
        DIFF_FOUND=1
        continue
    fi
    
    FILES_CHECKED=$((FILES_CHECKED + 1))
    
    # Compare files (ignoring timestamp comments)
    if ! diff -u \
        <(grep -v "// \tprotoc " "$EXISTING_FILE" | grep -v "^//\s*versions:" | grep -v "^//\s*\tprotoc-gen-go" | grep -v "^//\s*source:") \
        <(grep -v "// \tprotoc " "$GENERATED_FILE" | grep -v "^//\s*versions:" | grep -v "^//\s*\tprotoc-gen-go" | grep -v "^//\s*source:") \
        > /dev/null 2>&1; then
        echo "❌ Generated code out of date: pkg/proto/$proto_type/$proto_type.pb.go"
        echo "   Run: make proto-generate"
        DIFF_FOUND=1
    else
        echo "✅ $proto_type.pb.go is up-to-date"
    fi
done

echo ""
echo "📊 Summary: Checked $FILES_CHECKED generated files"

if [ $DIFF_FOUND -eq 1 ]; then
    echo ""
    echo "❌ Generated code is out of date!"
    echo ""
    echo "Fix by running:"
    echo "  make proto-generate"
    echo ""
    exit 1
fi

echo ""
echo "✅ All generated protobuf code is up-to-date!"
