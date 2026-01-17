# rule-updater – System Patterns

## Incremental snapshot updates
The service maintains a Redis snapshot of rules and updates it incrementally based on `rule.changed` events:
- **On startup**: Builds initial snapshot from all enabled rules in DB
- **On rule.changed event**: Loads current snapshot from Redis and applies incremental update:
  - **CREATED/UPDATED**: Fetches rule from DB and adds/updates it in snapshot
  - **DELETED/DISABLED**: Removes rule from snapshot
- Write updated snapshot to Redis and increment `rules:version` atomically

## Snapshot structure
- **Dictionaries**: Maps strings (severity/source/name) to integers for compression
- **Inverted indexes**: Maps severity/source/name values to lists of ruleInts
- **Rules map**: Maps ruleInt to (rule_id, client_id) for lookup
- **Rule integers**: Each rule gets a unique integer (ruleInt) used in indexes

## Update operations
- **AddRule**: Assigns new ruleInt, adds to dictionaries if needed, adds to indexes
- **UpdateRule**: Removes old rule and re-adds with new values (ensures consistency)
- **RemoveRule**: Searches through all indexes to find and remove ruleInt, cleans up empty index entries

This approach is more efficient than full rebuilds and maintains consistency with the database state.
