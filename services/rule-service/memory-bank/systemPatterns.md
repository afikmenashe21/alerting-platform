# rule-service – System Patterns

## Consistency expectations
- UI/API expects read-after-write for rule CRUD.
- Use optimistic locking with `version` on updates.

## Publish rule.changed
MVP: publish after DB commit.
Event contains:
- rule_id, client_id
- action: CREATED/UPDATED/DELETED/DISABLED
- new version + updated_at

(Outbox pattern can be added later.)
