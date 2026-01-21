# Protobuf Tooling and Validation

This document describes the enhanced protobuf tooling setup for the alerting platform, including linting, breaking change detection, and automated validation.

## Overview

The project uses a comprehensive protobuf tooling stack to ensure:
- **Correctness**: Proto definitions are syntactically valid
- **Best practices**: Code follows protobuf style guidelines
- **Compatibility**: Schema changes don't break existing consumers
- **Freshness**: Generated Go code stays synchronized with proto definitions

## Tools

### protoc (Protocol Buffer Compiler)
- **Purpose**: Compiles `.proto` files to Go code
- **Version**: 33.4
- **Usage**: `make proto-generate`

### protoc-gen-go (Go Plugin)
- **Purpose**: Go code generator plugin for protoc
- **Version**: v1.36.11
- **Usage**: Automatically invoked by protoc

### buf (Protobuf Tooling)
- **Purpose**: Linting, breaking change detection, and enhanced validation
- **Version**: 1.64.0
- **Features**:
  - Advanced linting with configurable rules
  - Breaking change detection against previous versions
  - Dependency management for proto imports
  - Alternative code generation (optional)

## Available Commands

### Core Commands

```bash
# Generate Go code from proto files
make proto-generate

# Validate proto syntax
make proto-validate

# Verify generated code is current
make proto-verify-generated
```

### Quality Checks

```bash
# Lint proto files (requires buf)
make proto-lint

# Check for breaking changes (requires buf)
make proto-breaking
```

### Utilities

```bash
# Check if dependencies are installed
make proto-check-deps

# Comprehensive verification
make proto-verify
```

## Configuration Files

### buf.yaml
Location: `proto/buf.yaml`

Configures buf linting and breaking change rules:

```yaml
version: v2
modules:
  - path: .
    lint:
      use:
        - STANDARD
      except:
        - DIRECTORY_SAME_PACKAGE    # Allow flat directory structure
        - PACKAGE_DIRECTORY_MATCH   # All protos in one directory
        - PACKAGE_SAME_DIRECTORY    # Multiple packages in same dir
        - PACKAGE_VERSION_SUFFIX    # No versioned packages
```

**Rationale for exceptions**:
- Our flat structure (all `.proto` files in `proto/`) is simpler for this monorepo
- Valid for protoc and works well for small-to-medium projects
- Easier to navigate and maintain than deeply nested directories

### buf.gen.yaml (Optional)
Location: `proto/buf.gen.yaml`

Alternative code generation configuration using buf instead of protoc directly. Currently unused, as the Makefile uses protoc for consistency with existing workflow.

## Verification Script

### verify-generated-code.sh
Location: `scripts/proto/verify-generated-code.sh`

Ensures generated code matches proto definitions:

1. Generates fresh code in temporary directory
2. Compares with existing generated files
3. Ignores timestamp/version comments
4. Exits with error if differences found

**Usage**:
```bash
make proto-verify-generated
```

**Output**:
```
✅ alerts.pb.go is up-to-date
✅ common.pb.go is up-to-date
✅ notifications.pb.go is up-to-date
✅ rules.pb.go is up-to-date
```

## CI/CD Integration

### GitHub Actions
Location: `.github/workflows/proto-validation.yml`

Automated validation on pull requests:
- Validates proto syntax
- Lints for best practices
- Verifies generated code is current
- Checks for breaking changes (warning only)

**Triggers**:
- Pull requests modifying `proto/**` or `pkg/proto/**`
- Pushes to main/master branch

### Pre-commit Hooks
Location: `.pre-commit-config.yaml`

Local validation before commits:

**Setup**:
```bash
pip install pre-commit
pre-commit install
```

**Hooks**:
- `proto-validate` - Validate proto syntax
- `proto-lint` - Lint with buf
- `proto-verify-generated` - Check generated code is current

**Manual run**:
```bash
pre-commit run --all-files
```

## Workflow

### Making Proto Changes

1. **Modify proto files** in `proto/` directory

2. **Generate code**:
   ```bash
   make proto-generate
   ```

3. **Validate changes**:
   ```bash
   make proto-validate       # Syntax check
   make proto-lint           # Best practices
   make proto-verify-generated  # Code freshness
   ```

4. **Check breaking changes**:
   ```bash
   make proto-breaking
   ```

5. **Commit both** proto files and generated code

### Schema Evolution Best Practices

1. **Never reuse field numbers**
   - Once assigned, field numbers are reserved forever
   - Use `reserved` keyword for removed fields

2. **Add new fields as optional**
   - Proto3 fields are optional by default
   - Old consumers can read new messages

3. **Deprecate before removing**
   - Mark fields as `deprecated = true`
   - Wait for all consumers to upgrade

4. **Check breaking changes**
   - Run `make proto-breaking` before merging
   - Review warnings carefully

5. **Test compatibility**
   - Old consumers should read new messages
   - New consumers should read old messages

### Example: Adding a New Field

```protobuf
// Before
message AlertNew {
  string alert_id = 1;
  Severity severity = 4;
  string source = 5;
  string name = 6;
}

// After - safe addition
message AlertNew {
  string alert_id = 1;
  Severity severity = 4;
  string source = 5;
  string name = 6;
  string priority = 7;  // New optional field
}
```

### Example: Removing a Field (Breaking Change)

```protobuf
// Before
message AlertNew {
  string alert_id = 1;
  int32 schema_version = 2;
  string deprecated_field = 3;
}

// After - mark as deprecated first
message AlertNew {
  string alert_id = 1;
  int32 schema_version = 2;
  string deprecated_field = 3 [deprecated = true];
}

// Later - reserve the number
message AlertNew {
  string alert_id = 1;
  int32 schema_version = 2;
  reserved 3;
  reserved "deprecated_field";
}
```

## Troubleshooting

### Generated Code Out of Date

**Error**:
```
❌ Generated code out of date: pkg/proto/alerts/alerts.pb.go
   Run: make proto-generate
```

**Solution**:
```bash
make proto-generate
git add pkg/proto/
```

### Buf Not Found

**Error**:
```
❌ buf not found. Install it first: brew install bufbuild/buf/buf
```

**Solution**:
```bash
# macOS
brew install bufbuild/buf/buf

# Linux
# See: https://buf.build/docs/installation
```

### Breaking Changes Detected

**Warning**:
```
⚠️  Breaking changes detected - review carefully
```

**Action**:
- Review the specific breaking changes
- Decide if they're intentional
- Document the breaking change in release notes
- Consider backwards-compatible alternatives

### Lint Failures

**Error**:
```
alerts.proto:5:1:Field name should be lower_snake_case
```

**Solution**:
- Fix the proto file to follow naming conventions
- Or add exception to `buf.yaml` if justified

## Benefits

### For Developers
- **Fast feedback**: Catch issues before code review
- **Consistency**: Automated style enforcement
- **Safety**: Breaking change detection prevents accidents
- **Confidence**: Verification ensures code is always current

### For Code Reviews
- **Reduced noise**: Style issues caught automatically
- **Focus**: Review logic, not formatting
- **Trust**: CI validates correctness

### For Operations
- **Reliability**: Schema changes are safe by default
- **Visibility**: Breaking changes are explicit
- **Documentation**: Proto files serve as contracts

## Future Enhancements

Potential improvements for future consideration:

1. **Buf Schema Registry (BSR)**
   - Centralized schema management
   - Version tracking and history
   - Cross-repository schema sharing

2. **Reflection API**
   - Dynamic message inspection
   - Runtime schema queries
   - Better debugging tools

3. **Custom Lint Rules**
   - Project-specific naming conventions
   - Domain-specific validation
   - Custom deprecation policies

4. **Generated Documentation**
   - HTML docs from proto comments
   - API reference generation
   - Change log automation

## References

- [Protocol Buffers Documentation](https://protobuf.dev/)
- [Buf Documentation](https://buf.build/docs/)
- [Protobuf Style Guide](https://protobuf.dev/programming-guides/style/)
- [Schema Evolution Best Practices](https://protobuf.dev/programming-guides/dos-donts/)
- [Project Proto Integration Strategy](./PROTOBUF_INTEGRATION_STRATEGY.md)
