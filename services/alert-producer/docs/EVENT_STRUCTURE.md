# Alert Event Structure

## Confirmed Structure

The alert-producer generates events with the following structure:

```json
{
  "alert_id": "550e8400-e29b-41d4-a716-446655440000",
  "schema_version": 1,
  "event_ts": 1705257600,
  "severity": "HIGH",
  "source": "api",
  "name": "timeout",
  "context": {
    "environment": "prod",
    "region": "us-east-1"
  }
}
```

## Field Specifications

### Required Fields

1. **alert_id** (string)
   - Type: UUID v4
   - Format: `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
   - Purpose: Unique identifier for the alert
   - Used as: Kafka message key for partition distribution

2. **schema_version** (integer)
   - Type: int
   - Current value: `1`
   - Purpose: Schema evolution support

3. **event_ts** (integer)
   - Type: int64
   - Format: Unix timestamp (seconds since epoch)
   - Purpose: When the alert was generated

4. **severity** (string, enum)
   - Type: string
   - Valid values: `LOW`, `MEDIUM`, `HIGH`
   - Purpose: Alert severity level
   - **Note**: CRITICAL is not used in this implementation

5. **source** (string)
   - Type: string
   - Examples: `"api"`, `"db"`, `"monitor"`, `"queue"`
   - Purpose: Identifies the source system that generated the alert

6. **name** (string)
   - Type: string
   - Examples: `"timeout"`, `"error"`, `"latency"`, `"memory"`
   - Purpose: Identifies the type/name of the alert

### Optional Fields

7. **context** (object)
   - Type: map[string]string
   - Purpose: Additional metadata
   - May include:
     - `environment`: `"prod"`, `"staging"`, `"dev"` (30% probability)
     - `region`: `"us-east-1"`, `"us-west-2"`, `"eu-west-1"` (20% probability)
   - Can be empty: `{}` or omitted entirely

## Kafka Message Format

- **Topic**: `alerts.new`
- **Key**: `alert_id` (byte array)
- **Value**: JSON-encoded alert (byte array)
- **Headers**:
  - `schema_version`: String representation of schema version
  - `severity`: Severity value for filtering/routing
- **Timestamp**: Set from `event_ts` field

## Validation

The event structure matches the requirements:
- ✅ **severity**: Enum of LOW, MEDIUM, HIGH
- ✅ **source**: String
- ✅ **name**: String
- ✅ All required fields present
- ✅ JSON format
- ✅ Properly keyed for Kafka partition distribution
