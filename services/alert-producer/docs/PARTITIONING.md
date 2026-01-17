# Partitioning Strategy

## Partition Key

The alert-producer uses **key-based partitioning** to ensure even distribution across Kafka partitions and avoid hot partitions.

## Implementation

1. **Partition Key**: SHA-256 hash of `alert_id` (first 16 bytes)
   - Deterministic: same `alert_id` → same partition
   - Even distribution: random UUIDs hash evenly across partitions
   - Prevents hot partitions

2. **Balancer**: `kafka.Hash{}`
   - Uses the message key to determine partition
   - Kafka internally hashes the key to select partition
   - Ensures even distribution across all partitions

## Why Hash the alert_id?

- **UUIDs are random**: While UUIDs provide good distribution, explicit hashing gives us:
  - Guaranteed even distribution regardless of UUID version
  - Explicit control over partitioning logic
  - Protection against future UUID format changes

- **Deterministic**: Same `alert_id` always maps to the same partition
  - Useful for ordering guarantees (if needed in future)
  - Helps with debugging and tracing

- **Avoids hot partitions**: Even if UUIDs had patterns, hashing ensures uniform distribution

## Partition Selection

```
alert_id (UUID) → SHA-256 hash → Kafka Hash balancer → Partition (0-2)
```

Example:
- `alert_id: "abc-123-def"` → hash → partition 1
- `alert_id: "xyz-789-ghi"` → hash → partition 0
- `alert_id: "abc-123-def"` → hash → partition 1 (same as before)

## Configuration

- **Topic**: `alerts.new`
- **Partitions**: 3
- **Replication Factor**: 1
- **Partitioning**: Key-based (hashed `alert_id`)

## Verification

To verify even distribution, check partition distribution:

```bash
# Check message distribution across partitions
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic alerts.new
```

You should see roughly equal message counts across all partitions.
