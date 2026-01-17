# Wildcard Rules Usage Guide

## Overview

Wildcard support allows rules to match any value for a field by using `"*"` as a sentinel value. This enables:
- Multiple rules per client matching the same alert
- Testing the grouping mechanism
- More flexible rule definitions

## How It Works

### Wildcard Syntax

- `"*"` in any field means "match any value"
- At least one field must be non-wildcard (cannot have all three as `"*"`)

### Examples

**Rule 1:** `LOW/test-source/test-name` (exact match)
**Rule 2:** `*/test-source/test-name` (matches any severity)
**Rule 3:** `LOW/*/test-name` (matches any source)
**Rule 4:** `LOW/test-source/*` (matches any name)

**Alert:** `LOW/test-source/test-name`

**Result:** Matches all 4 rules → Evaluator groups by `client_id` → One message with `rule_ids: [rule1, rule2, rule3, rule4]`

## Creating Wildcard Rules

### Via API

```bash
# Create rule with wildcard severity
curl -X POST http://localhost:8081/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "afik-test",
    "severity": "*",
    "source": "test-source",
    "name": "test-name"
  }'

# Create rule with wildcard source
curl -X POST http://localhost:8081/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "afik-test",
    "severity": "LOW",
    "source": "*",
    "name": "test-name"
  }'
```

### Via UI

In the rule-service-ui, you can enter `"*"` directly in the severity, source, or name fields.

## Testing Multiple Rules Per Client

### Step 1: Create Multiple Rules

Create rules for client `afik-test`:

1. **Rule 1 (exact):** `LOW/test-source/test-name`
2. **Rule 2 (wildcard severity):** `*/test-source/test-name`
3. **Rule 3 (wildcard source):** `LOW/*/test-name`
4. **Rule 4 (wildcard name):** `LOW/test-source/*`

### Step 2: Generate Test Alert

```bash
# Generate test alert that matches all rules
make run-test ARGS="-burst 1"
```

This generates: `LOW/test-source/test-name`

### Step 3: Verify Matching

Check evaluator logs - you should see:
```
Published matched alert
  alert_id: ...
  client_id: afik-test
  rule_ids: [rule-1-id, rule-2-id, rule-3-id, rule-4-id]
```

### Step 4: Verify Deduplication

The aggregator will:
1. Receive one message with all 4 rule_ids
2. Create one notification (dedupe at `client_id + alert_id` level)
3. Include all rule_ids in the notification for explainability

## Database Migration

Run the migration to enable wildcard support:

```bash
# From root directory
make run-migrations
```

This adds migration `000007_allow_wildcard_in_rules.up.sql` which:
- Updates the CHECK constraint to allow `"*"` as a valid severity value
- Allows `"*"` in source and name fields (no constraint changes needed)

## Constraints

1. **Unique Constraint:** The unique constraint `(client_id, severity, source, name)` still applies
   - You cannot create two identical rules (even with wildcards)
   - But you can have multiple different wildcard combinations

2. **All Wildcards:** Cannot create a rule with all three fields as `"*"`

3. **Severity Values:** `"*"` is only valid for severity, source, and name fields

## Implementation Details

### Evaluator Matching Logic

The evaluator:
1. Gets exact matches for each field
2. Also gets wildcard matches (`"*"` bucket)
3. Combines them before intersection
4. Groups results by `client_id`

### Snapshot Building

Rules with `"*"` are added to the `"*"` bucket in the indexes:
- `bySeverity["*"]` contains all rules with wildcard severity
- `bySource["*"]` contains all rules with wildcard source
- `byName["*"]` contains all rules with wildcard name

### Performance

Wildcard matching adds minimal overhead:
- One additional lookup per field (checking `"*"` bucket)
- List combination is O(n) where n is the number of wildcard rules
- Intersection algorithm remains efficient

## Example Test Scenario

**Goal:** Test that one alert matches multiple rules for the same client

1. **Create Rules:**
   ```bash
   # Rule 1: Exact match
   curl -X POST http://localhost:8081/api/v1/rules \
     -d '{"client_id":"afik-test","severity":"LOW","source":"test-source","name":"test-name"}'
   
   # Rule 2: Wildcard severity
   curl -X POST http://localhost:8081/api/v1/rules \
     -d '{"client_id":"afik-test","severity":"*","source":"test-source","name":"test-name"}'
   
   # Rule 3: Wildcard source
   curl -X POST http://localhost:8081/api/v1/rules \
     -d '{"client_id":"afik-test","severity":"LOW","source":"*","name":"test-name"}'
   ```

2. **Generate Alert:**
   ```bash
   make run-test ARGS="-burst 1"
   ```

3. **Verify:**
   - Check evaluator logs for `rule_ids` array with 3 rule IDs
   - Check aggregator creates one notification
   - Check notification has all 3 rule_ids

## Troubleshooting

**Issue:** Rule creation fails with "severity must be one of..."
- **Solution:** Ensure you're using `"*"` (with quotes) not `*` (without quotes)

**Issue:** Unique constraint violation
- **Solution:** You're trying to create a duplicate rule. Wildcards still need unique combinations.

**Issue:** Alert doesn't match wildcard rule
- **Solution:** 
  1. Verify rule was created successfully
  2. Check rule-updater has rebuilt the snapshot
  3. Check evaluator has reloaded the snapshot (check version polling)
