# rule-updater – Project Brief

## Purpose
Build and maintain a Redis snapshot of rules for evaluator warm start and fast reload.

## Inputs
- Kafka `rule.changed` (trigger)
- Postgres rules table (source of truth)

## Outputs
- Redis `rules:snapshot` (serialized indexes + dictionaries)
- Redis `rules:version` (monotonic integer)
