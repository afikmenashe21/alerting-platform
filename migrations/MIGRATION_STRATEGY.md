# Database Migration Strategy

## Overview

Multiple services share the same PostgreSQL database (`alerting`). To prevent version conflicts, we use a **coordinated migration numbering system**.

## Migration Ownership & Versioning

### Version Ranges by Service

| Service | Migration Range | Tables Owned |
|---------|----------------|--------------|
| `rule-service` | 000001 - 000005 | `clients`, `rules`, `endpoints` |
| `aggregator` | 000006+ | `notifications` |
| `sender` | (future) | (future tables) |

### Current Migrations

**rule-service (000001-000005):**
- `000001` - Create clients table
- `000002` - Create rules table
- `000003` - Create endpoints table
- `000004` - Remove email from rules
- `000005` - Remove email from clients

**aggregator (000006+):**
- `000006` - Create notifications table

## Rules for Creating New Migrations

### 1. Check Current Highest Version
Before creating a new migration, check what the highest migration number is across all services:

```bash
# Find highest migration number
find . -name "*.up.sql" -path "*/migrations/*" | sed 's/.*\/\([0-9]*\)_.*/\1/' | sort -n | tail -1
```

### 2. Assign Next Sequential Number
- If highest is `000006`, your new migration should be `000007`
- Always increment by 1
- Never reuse numbers

### 3. Service Ownership
- **rule-service**: Owns control-plane tables (clients, rules, endpoints)
- **aggregator**: Owns data-plane tables (notifications)
- **sender**: (future) May own delivery tracking tables

### 4. Cross-Service Dependencies
If a migration in one service needs to reference tables from another:
- Document the dependency clearly in the migration file
- Ensure the dependent service's migrations run first
- Use `IF NOT EXISTS` clauses when possible

## Migration Workflow

### Creating a New Migration

1. **Determine ownership**: Which service owns the table being modified?
2. **Check version**: Run `make check-migrations` to see current state
3. **Create migration**: Use the service's Makefile:
   ```bash
   cd <service-directory>
   make migrate-create NAME=description_of_change
   ```
4. **Update this document**: Add the new migration to the version table above
5. **Test**: Run migrations up and down to verify

### Running Migrations

Each service can run migrations independently, but they all target the same database:

```bash
# From rule-service
cd rule-service && make migrate-up

# From aggregator  
cd aggregator && make migrate-up
```

**Important**: Migrations are idempotent - running them multiple times is safe (uses `IF NOT EXISTS`).

## Validation & Consistency Checks

### Check Migration Consistency

Run the validation script to ensure all services are aligned:

```bash
make check-migrations
```

This will:
- List all migrations across services
- Check for version conflicts
- Verify migration files are paired (up/down)
- Show current database version

### Manual Check

```bash
# Check database version
docker exec <postgres-container> psql -U postgres -d alerting -c "SELECT version, dirty FROM schema_migrations;"

# List all migration files
find . -path "*/migrations/*.sql" | sort
```

## Troubleshooting

### Version Conflict Error

If you see: `error: no migration found for version X`

**Cause**: Database thinks it's at version X, but no migration file exists for that version.

**Solution**:
1. Check what migrations exist: `find . -path "*/migrations/*.sql" | sort`
2. Check database version: `docker exec <postgres> psql -U postgres -d alerting -c "SELECT version FROM schema_migrations;"`
3. If database version is higher than files, you may need to:
   - Manually fix the schema_migrations table, OR
   - Add the missing migration file

### Migration Out of Order

If migrations are applied out of order, the database version may be inconsistent.

**Solution**: Always run migrations sequentially. If out of order, you may need to:
1. Rollback to a known good state
2. Re-apply migrations in order
3. Or manually adjust the schema_migrations table

## Best Practices

1. **Always use sequential numbering** - Never skip numbers
2. **Document dependencies** - If migration depends on another service's tables
3. **Use IF NOT EXISTS** - Makes migrations idempotent
4. **Test rollback** - Always test `migrate-down` works
5. **Commit together** - If multiple services need migrations, coordinate the PR
6. **Update this doc** - Keep the version table current

## Future Considerations

- **Shared migrations directory**: Could consolidate all migrations into a single `migrations/` folder at root
- **Migration service**: Dedicated service that owns all migrations
- **Schema registry**: Track schema changes in a registry
- **Automated validation**: CI/CD check that validates migration consistency
