# Generate Test Data Script

This script cleans the database and generates 100 clients with rules and endpoints for testing.

## Usage

### Using Make (Recommended)
```bash
make generate-test-data
```

### Using the script directly
```bash
./scripts/test-data/generate-test-data.sh
```

### With custom database connection
```bash
POSTGRES_DSN="postgres://user:pass@host:port/dbname?sslmode=disable" ./scripts/test-data/generate-test-data.sh
```

Or:
```bash
./scripts/test-data/generate-test-data.sh "postgres://user:pass@host:port/dbname?sslmode=disable"
```

## What it does

1. **Cleans the database**: Deletes all existing data from:
   - `endpoints` table
   - `rules` table
   - `notifications` table
   - `clients` table

2. **Generates 100 clients**:
   - Client IDs: `client-001` through `client-100`
   - Client names: `Client 1` through `Client 100`

3. **Generates rules** (1-5 rules per client, randomly distributed):
   - Severity: LOW, MEDIUM, HIGH, CRITICAL (random)
   - Source: api, db, cache, monitor, queue, worker, frontend, backend (random)
   - Name: timeout, error, crash, slow, memory, cpu, disk, network, auth, validation (random)

4. **Generates endpoints** (1-3 endpoints per rule, randomly distributed):
   - Types: email, webhook, slack
   - Email: `alert-XXX-Y@example.com`
   - Webhook: `https://webhook.example.com/client-XXX/rule-YYYYYYYY`
   - Slack: `#alerts-client-XXX`
   - Avoids duplicate endpoint types per rule

## Requirements

- Go 1.22+
- PostgreSQL database with migrations run
- `github.com/lib/pq` package (automatically downloaded)

## Example Output

```
Connecting to database...
Cleaning database...
Generating 100 clients with rules and endpoints...
Progress: 10 clients, 32 rules, 78 endpoints created...
Progress: 20 clients, 65 rules, 156 endpoints created...
...
=== Generation Complete ===
Clients created: 100
Rules created: 300
Endpoints created: 750
Average rules per client: 3.00
Average endpoints per rule: 2.50
```

## Notes

- The script uses `ON CONFLICT DO NOTHING` to handle any duplicate entries gracefully
- All rules are created with `enabled = TRUE`
- All endpoints are created with `enabled = TRUE`
- The script respects foreign key constraints when cleaning
