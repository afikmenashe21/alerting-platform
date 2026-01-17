# alert-producer – Active Context

## Current Status
Service implementation complete. Ready for testing with Kafka.

## Structure
- `cmd/alert-producer/main.go` - CLI entry point with flags and main loop
- `internal/config/` - Configuration parsing and validation
- `internal/generator/` - Alert generation with weighted distributions
- `internal/producer/` - Kafka producer wrapper

## Recent Changes
- Updated default distributions to match test-data generator values
- Alerts now use the same severity/source/name values as rules in the database
- Boilerplate alert changed to HIGH/api/timeout (common rule combination)

## Next Steps
- Test with local Kafka instance
- Verify message format matches evaluator expectations
- Load testing with various RPS values
- Verify alerts match existing rules and trigger notifications
