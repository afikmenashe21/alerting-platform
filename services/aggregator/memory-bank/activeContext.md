# aggregator – Active Context

## Completed
- ✅ Database migrations for notifications table with unique constraint
- ✅ Consumer for `alerts.matched` topic
- ✅ Idempotent insert logic with `ON CONFLICT DO NOTHING RETURNING`
- ✅ Producer for `notifications.ready` topic
- ✅ Main processing loop with correct offset commit ordering
- ✅ Configuration, Makefile, and README
- ✅ Uses centralized infrastructure (managed at root level)
- ✅ Code cleanup and modularization:
  - Extracted shared Kafka validation helpers and constants
  - Extracted message building and JSON marshaling helpers
  - Added event builder helper functions
  - Removed duplicate validation logic
  - All tests pass; behavior unchanged

## Architecture
- Consumes `alerts.matched` (one message per client_id)
- Inserts into `notifications` table idempotently
- Emits `notifications.ready` only for newly created notifications
- Commits offsets after successful persistence
