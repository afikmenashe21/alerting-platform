# Wildcard/NULL Field Support for Rules

This document describes how to implement wildcard/NULL field support to allow multiple rules to match the same alert.

## Goal

Allow rules to have NULL/wildcard fields so that:
- Rule 1: `LOW/test-source/test-name` (exact match)
- Rule 2: `NULL/test-source/test-name` (matches any severity)
- Rule 3: `LOW/NULL/test-name` (matches any source)

An alert `LOW/test-source/test-name` would match all three rules.

## Current Limitations

1. **Database Schema**: All fields are `NOT NULL`
2. **Unique Constraint**: `(client_id, severity, source, name)` - NULL values complicate this
3. **Matching Logic**: Uses exact match only
4. **Indexes**: Built for exact matches only

## Implementation Approach

### Option 1: Use Sentinel Value (Recommended for MVP)

Use a special sentinel value like `"*"` or `""` to represent "match any" instead of NULL.

**Pros:**
- No database schema changes needed
- Simpler to implement
- Works with existing unique constraint

**Cons:**
- Not as intuitive as NULL
- Requires validation to prevent users from creating rules with "*" as actual value

### Option 2: Allow NULL Fields

Allow NULL in database and treat NULL as "match any".

**Pros:**
- More intuitive
- Standard SQL pattern

**Cons:**
- Requires schema migration
- Unique constraint needs special handling (NULL != NULL in SQL)
- More complex matching logic

## Recommended Implementation (Option 1: Sentinel Value)

### Step 1: Database Schema

Keep current schema, but allow `"*"` as a special value:

```sql
-- No migration needed, but add validation
-- severity, source, name can be "*" to mean "match any"
```

### Step 2: Update Unique Constraint

Modify unique constraint to handle wildcards:

```sql
-- Current constraint
CONSTRAINT rules_client_criteria_unique UNIQUE (client_id, severity, source, name)

-- New approach: Allow multiple rules with wildcards
-- But ensure at most one exact match rule per client
-- This requires application-level validation or a more complex constraint
```

### Step 3: Update Matching Logic

Modify `indexes.Match()` to handle wildcards:

```go
func (idx *Indexes) Match(severity, source, name string) map[string][]string {
    // Get candidate lists for each field
    severityRules := idx.bySeverity[severity]
    sourceRules := idx.bySource[source]
    nameRules := idx.byName[name]
    
    // Also include wildcard rules
    wildcardSeverityRules := idx.bySeverity["*"]  // Rules that match any severity
    wildcardSourceRules := idx.bySource["*"]      // Rules that match any source
    wildcardNameRules := idx.byName["*"]          // Rules that match any name
    
    // Combine exact matches with wildcards
    allSeverityRules := combine(severityRules, wildcardSeverityRules)
    allSourceRules := combine(sourceRules, wildcardSourceRules)
    allNameRules := combine(nameRules, wildcardNameRules)
    
    // Continue with intersection logic...
}
```

### Step 4: Update Snapshot Building

In `rule-updater`, when building snapshots, include rules with wildcards in special indexes:

```go
// In snapshot builder
if rule.Severity == "*" {
    // Add to all severity buckets (or special "*" bucket)
}
```

### Step 5: Update Rule Service Validation

Add validation to prevent invalid combinations:

```go
// Validate rule creation
if severity == "*" && source == "*" && name == "*" {
    return error("Cannot create rule that matches everything")
}
```

## Testing Multiple Rules Per Client

With wildcard support, you can create:

**For client `afik-test`:**
1. Rule 1: `LOW/test-source/test-name` (exact)
2. Rule 2: `*/test-source/test-name` (any severity)
3. Rule 3: `LOW/*/test-name` (any source)
4. Rule 4: `LOW/test-source/*` (any name)

**Alert:** `LOW/test-source/test-name`

**Matches:** All 4 rules → Evaluator groups by `client_id` → One message with `rule_ids: [rule1, rule2, rule3, rule4]`

## Alternative: Quick Test Without Full Implementation

For quick testing without implementing full wildcard support, you can:

1. **Create rules with slightly different criteria** that all match the same alert pattern
2. **Use the test data generator** to create multiple rules for the same client with overlapping patterns
3. **Manually create rules** via the API with different combinations

Example for testing:
- Rule 1: `LOW/test-source/test-name`
- Rule 2: `LOW/test-source/error` (different name, but send alert that could match both if we had OR logic)
- Rule 3: `MEDIUM/test-source/test-name` (different severity)

But this won't work with exact match - you'd need the alert to match multiple exact rules, which requires wildcards.

## Recommendation

For MVP testing, the simplest approach is:
1. **Use multiple endpoints per rule** to test multiple notification destinations
2. **Implement basic wildcard support** using `"*"` sentinel value (Option 1)
3. **Document the limitation** that exact match MVP doesn't support multiple rules per alert

If you want to proceed with wildcard support, I can implement Option 1 (sentinel value approach) which requires minimal schema changes.
