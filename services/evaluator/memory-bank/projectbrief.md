# evaluator – Project Brief

## Purpose
Consume `alerts.new`, evaluate rules (exact match for MVP), and emit grouped matches to `alerts.matched`.

## Input
Topic: `alerts.new`
Fields used for matching:
- severity (enum)
- source (string)
- name (string)
Optional: context json, event_ts.

## Output
Topic: `alerts.matched`
**One message per client_id** (if alert matches multiple clients, multiple messages are published):
- alert_id
- alert payload (severity/source/name/context)
- client_id (the client this message is for)
- rule_ids[] (all rule IDs that matched for this client)

Messages are keyed by `client_id` for tenant locality (partitioning).

## Key properties
- Stateless w.r.t dedupe (duplicates tolerated).
- Very fast evaluation using in-memory rule indexes.
- Warm start from Redis snapshot.
