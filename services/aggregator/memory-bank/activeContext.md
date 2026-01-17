# aggregator – Active Context

## Completed
- ✅ Database migrations for notifications table with unique constraint
- ✅ Consumer for `alerts.matched` topic
- ✅ Idempotent insert logic with `ON CONFLICT DO NOTHING RETURNING`
- ✅ Producer for `notifications.ready` topic
- ✅ Main processing loop with correct offset commit ordering
- ✅ Configuration, Makefile, and README
- ✅ Uses centralized infrastructure (managed at root level)

## Architecture
- Consumes `alerts.matched` (one message per client_id)
- Inserts into `notifications` table idempotently
- Emits `notifications.ready` only for newly created notifications
- Commits offsets after successful persistence
