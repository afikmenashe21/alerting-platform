# aggregator – System Patterns

## Dedupe mechanism
Unique key: (client_id, alert_id). One row = one notification per client per alert.

SQL:
- INSERT ... ON CONFLICT DO NOTHING RETURNING notification_id
- If returning row exists -> emit notifications.ready
- Else -> skip emit (already processed)

## Offset commit rule
Commit Kafka offsets only after DB transaction completes successfully.
