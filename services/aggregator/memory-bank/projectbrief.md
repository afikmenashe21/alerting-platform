# aggregator – Project Brief

## Purpose
Consume `alerts.matched`, persist notification intents with dedupe, and emit `notifications.ready` for sending.

## Why it exists
Kafka is at-least-once; evaluator can emit duplicates. Aggregator provides the **idempotency boundary**.

## Input
Topic: `alerts.matched`
Each message contains matches grouped by client.

## Output
Topic: `notifications.ready`
Emit work items only for newly created notification rows:
- notification_id
- client_id
- alert_id
