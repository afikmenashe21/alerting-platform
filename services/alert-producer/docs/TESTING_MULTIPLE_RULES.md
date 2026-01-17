# Testing Multiple Rules Per Client

This guide explains how to test the scenario where a client has multiple rules that match the same alert, and verify the grouping and deduplication mechanisms.

## Overview

When an alert matches multiple rules for the same client:
1. **Evaluator** groups by `client_id` and includes all matching `rule_ids[]` in one message
2. **Aggregator** deduplicates at the `(client_id, alert_id)` level
3. One notification is created per client per alert, but it contains all matching rule IDs

## Setup

### Step 1: Create Multiple Rules for the Same Client

Create multiple rules for client `afik-test` that all match the same alert pattern:

**Rule 1:**
- Client: `afik-test`
- Severity: `LOW`
- Source: `test-source`
- Name: `test-name`

**Rule 2:**
- Client: `afik-test`
- Severity: `LOW`
- Source: `test-source`
- Name: `test-name`

**Note:** The database has a unique constraint on `(client_id, severity, source, name)`, so you cannot create two identical rules. Instead, you have two options:

#### Option A: Create Rules with Different Endpoints
Create one rule with multiple endpoints. The rule will match, and all endpoints will be used.

#### Option B: Test with Different Alert Patterns
Create multiple rules for the same client that match different alerts, then send alerts that match multiple of those rules.

**Example:**
- Rule 1: `LOW/test-source/test-name`
- Rule 2: `LOW/test-source/error` (different name)
- Rule 3: `MEDIUM/test-source/test-name` (different severity)

Then send an alert that matches multiple rules (e.g., if you have a rule that matches `LOW/test-source/*` pattern - but this requires wildcard support which isn't in MVP).

### Step 2: Generate Test Alerts

Use test mode to generate varied alerts with one test alert:

```bash
# Generate 10 alerts (first is test alert, rest are varied)
make run-test ARGS="-burst 10"

# Or continuous mode
make run-test ARGS="-rps 2 -duration 30s"
```

## Expected Behavior

### Evaluator Output

When the test alert (LOW/test-source/test-name) matches your rule(s), the evaluator will publish to `alerts.matched`:

```json
{
  "alert_id": "...",
  "severity": "LOW",
  "source": "test-source",
  "name": "test-name",
  "client_id": "afik-test",
  "rule_ids": ["rule-id-1", "rule-id-2", ...]  // All matching rule IDs
}
```

**Key Points:**
- One message per `client_id` (even if multiple rules match)
- All matching `rule_ids[]` are included in the array
- If the same alert matches multiple clients, multiple messages are published (one per client)

### Aggregator Behavior

The aggregator will:
1. Receive the `alerts.matched` message
2. Insert into `notifications` table with unique constraint on `(client_id, alert_id)`
3. If the same alert is processed twice (Kafka redelivery), the insert will fail due to unique constraint → **deduplication works**
4. Publish `notifications.ready` event

### Testing Deduplication

To test deduplication:
1. Send the same alert twice (same `alert_id`)
2. Or let Kafka redeliver the same message
3. Verify only one notification is created in the database
4. Check that the aggregator logs show the duplicate was handled

## Verification Steps

1. **Check Evaluator Logs:**
   ```bash
   # Look for: "Published matched alert" with rule_ids array
   # Should see: "rule_ids": ["rule-1", "rule-2"] if multiple rules match
   ```

2. **Check Aggregator Logs:**
   ```bash
   # Look for: "Created notification" or "Duplicate notification skipped"
   # Should see deduplication working if same alert processed twice
   ```

3. **Check Database:**
   ```sql
   -- Check notifications table
   SELECT * FROM notifications WHERE client_id = 'afik-test';
   
   -- Should see one row per unique (client_id, alert_id) combination
   ```

4. **Check Notifications Ready Topic:**
   ```bash
   # Consume from notifications.ready topic
   # Should see one message per notification
   ```

## Example Test Scenario

1. **Create Rule:**
   - Client: `afik-test`
   - Severity: `LOW`
   - Source: `test-source`
   - Name: `test-name`
   - Endpoints: Add multiple email endpoints

2. **Generate Alerts:**
   ```bash
   make run-test ARGS="-burst 5"
   ```

3. **Verify:**
   - Evaluator matches the test alert
   - Aggregator creates one notification per unique alert
   - Sender processes the notification
   - All endpoints receive the notification

## Notes

- The unique constraint on `(client_id, severity, source, name)` prevents creating identical rules
- **Wildcard support is now implemented!** Use `"*"` to match any value for a field
- The deduplication happens at the aggregator level, not the evaluator level
- Multiple endpoints per rule is the way to test multiple notification destinations

## Wildcard Support (✅ Implemented)

Wildcard support is now available! Use `"*"` to mean "match any" for a field.

**Example with wildcards:**
- Rule 1: `LOW/test-source/test-name` (exact)
- Rule 2: `*/test-source/test-name` (any severity)
- Rule 3: `LOW/*/test-name` (any source)
- Rule 4: `LOW/test-source/*` (any name)

Alert `LOW/test-source/test-name` matches all four rules, and the evaluator groups them by `client_id` with all `rule_ids` in one message.

**To use:**
1. Run migration: `make run-migrations`
2. Create rules with `"*"` in any field (severity, source, or name)
3. Generate test alerts: `make run-test`

See `docs/WILDCARD_RULES_USAGE.md` for full usage guide.
