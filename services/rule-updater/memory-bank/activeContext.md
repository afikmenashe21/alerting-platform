# rule-updater – Active Context

## Status: ✅ Complete

The rule-updater service is fully implemented and ready for use.

## What it does
- Consumes `rule.changed` events from Kafka
- On startup: Builds initial snapshot from all enabled rules in Postgres
- On rule change: Loads current snapshot from Redis and applies incremental update:
  - **CREATED/UPDATED**: Fetches rule from DB and adds/updates it in snapshot
  - **DELETED/DISABLED**: Removes rule from snapshot
- Writes updated snapshot to Redis at `rules:snapshot` key
- Increments version at `rules:version` key atomically

## Key design decisions
- **Incremental updates**: Updates snapshot incrementally based on rule.changed events instead of full rebuilds
- **Load from Redis**: Loads current snapshot before applying updates to maintain consistency
- **Atomic snapshot + version update**: Uses Redis pipeline to update both together
- **At-least-once semantics**: Commits Kafka offset only after successful snapshot write
- **Snapshot format**: Matches evaluator's expected format exactly
- **Update strategy**: For UPDATED rules, removes old entry and re-adds with new values (ensures index consistency)
