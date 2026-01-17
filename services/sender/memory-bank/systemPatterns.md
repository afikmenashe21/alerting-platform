# sender – System Patterns

## Idempotent send best-effort
- If row status already SENT -> no-op.
- Otherwise attempt send and then update status.

## Failure note
If email sent but status update fails, a retry can cause duplicate email.
Mitigation later: provider idempotency key / message-id.
