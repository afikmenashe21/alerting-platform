#!/bin/bash
# Comprehensive verification script for protobuf setup

set -e

echo "🔍 Protobuf Setup Verification"
echo "=============================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

ERRORS=0
WARNINGS=0

# Function to check command
check_command() {
    if command -v "$1" &> /dev/null; then
        echo -e "${GREEN}✅${NC} $1 found"
        return 0
    else
        echo -e "${RED}❌${NC} $1 not found"
        ((ERRORS++))
        return 1
    fi
}

# Function to check file/directory exists
check_path() {
    if [ -e "$1" ]; then
        echo -e "${GREEN}✅${NC} $1 exists"
        return 0
    else
        echo -e "${RED}❌${NC} $1 not found"
        ((ERRORS++))
        return 1
    fi
}

# Function to check file contains text
check_file_contains() {
    if grep -q "$2" "$1" 2>/dev/null; then
        echo -e "${GREEN}✅${NC} $1 contains: $2"
        return 0
    else
        echo -e "${RED}❌${NC} $1 missing: $2"
        ((ERRORS++))
        return 1
    fi
}

echo "1. Checking Dependencies"
echo "------------------------"
check_command "protoc"
if check_command "protoc-gen-go"; then
    PROTOC_GEN_GO_PATH=$(which protoc-gen-go)
    echo "   Location: $PROTOC_GEN_GO_PATH"
fi

if command -v buf &> /dev/null; then
    echo -e "${GREEN}✅${NC} buf found (optional)"
else
    echo -e "${YELLOW}⚠️${NC}  buf not found (optional, for linting)"
    ((WARNINGS++))
fi
echo ""

echo "2. Checking Proto Definition Files"
echo "-----------------------------------"
check_path "proto/common.proto"
check_path "proto/alerts.proto"
check_path "proto/rules.proto"
check_path "proto/notifications.proto"
check_path "proto/README.md"
echo ""

echo "3. Validating Proto Files"
echo "-------------------------"
if command -v protoc &> /dev/null; then
    echo "Validating proto files..."
    if protoc --proto_path=proto --descriptor_set_out=/dev/null proto/*.proto 2>&1 | grep -q "error"; then
        echo -e "${RED}❌${NC} Proto validation failed"
        protoc --proto_path=proto --descriptor_set_out=/dev/null proto/*.proto 2>&1 | head -10
        ((ERRORS++))
    else
        echo -e "${GREEN}✅${NC} All proto files are valid"
    fi
else
    echo -e "${YELLOW}⚠️${NC}  Skipping validation (protoc not found)"
    ((WARNINGS++))
fi
echo ""

echo "4. Checking Generated Code"
echo "--------------------------"
if [ -d "pkg/proto" ]; then
    echo -e "${GREEN}✅${NC} pkg/proto directory exists"
    
    # Check for generated packages
    check_path "pkg/proto/common"
    check_path "pkg/proto/alerts"
    check_path "pkg/proto/rules"
    check_path "pkg/proto/notifications"
    
    # Check for .pb.go files
    PB_FILES=$(find pkg/proto -name "*.pb.go" 2>/dev/null | wc -l | tr -d ' ')
    if [ "$PB_FILES" -gt 0 ]; then
        echo -e "${GREEN}✅${NC} Found $PB_FILES generated .pb.go files"
    else
        echo -e "${RED}❌${NC} No .pb.go files found. Run: make proto-generate"
        ((ERRORS++))
    fi
    
    # Check go.mod exists
    check_path "pkg/proto/go.mod"
else
    echo -e "${RED}❌${NC} pkg/proto directory not found. Run: make proto-generate"
    ((ERRORS++))
fi
echo ""

echo "5. Checking Build Tooling"
echo "-------------------------"
check_file_contains "Makefile" "proto-generate"
check_file_contains "Makefile" "proto-validate"
check_file_contains "Makefile" "proto-check-deps"
check_path "scripts/proto/check-proto-deps.sh"
echo ""

echo "6. Testing Code Generation (Dry Run)"
echo "--------------------------------------"
if command -v protoc &> /dev/null && command -v protoc-gen-go &> /dev/null; then
    echo "Testing protoc command..."
    TEMP_DIR=$(mktemp -d)
    if protoc \
        --go_out="$TEMP_DIR" \
        --go_opt=paths=source_relative \
        -Iproto \
        proto/common.proto 2>&1 | grep -q "error"; then
        echo -e "${RED}❌${NC} Code generation test failed"
        protoc --go_out="$TEMP_DIR" --go_opt=paths=source_relative -Iproto proto/common.proto 2>&1
        ((ERRORS++))
    else
        echo -e "${GREEN}✅${NC} Code generation test passed"
    fi
    rm -rf "$TEMP_DIR"
else
    echo -e "${YELLOW}⚠️${NC}  Skipping generation test (dependencies not found)"
    ((WARNINGS++))
fi
echo ""

echo "7. Checking Documentation"
echo "-------------------------"
check_path "docs/architecture/PROTOBUF_INTEGRATION_STRATEGY.md"
echo ""

echo "8. Testing Go Module Structure"
echo "------------------------------"
if [ -f "pkg/proto/go.mod" ]; then
    echo "Checking proto package go.mod..."
    if grep -q "google.golang.org/protobuf" pkg/proto/go.mod; then
        echo -e "${GREEN}✅${NC} Proto package has protobuf dependency"
    else
        echo -e "${RED}❌${NC} Proto package missing protobuf dependency"
        ((ERRORS++))
    fi
else
    echo -e "${YELLOW}⚠️${NC}  Proto package go.mod not found (will be created on first generation)"
    ((WARNINGS++))
fi
echo ""

echo "=============================="
echo "Summary"
echo "=============================="

if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}✅ All critical checks passed!${NC}"
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}⚠️  $WARNINGS warning(s) (non-critical)${NC}"
    fi
    echo ""
    echo "Next steps:"
    echo "  1. If code not generated yet: make proto-generate"
    echo "  2. Start migrating services (producer/consumer code changes)"
    exit 0
else
    echo -e "${RED}❌ $ERRORS error(s) found${NC}"
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}⚠️  $WARNINGS warning(s)${NC}"
    fi
    echo ""
    echo "Fix errors and run again."
    exit 1
fi
