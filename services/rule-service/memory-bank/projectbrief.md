# rule-service – Project Brief

## Purpose
Control-plane API for managing clients and rules.
Persists to Postgres and publishes `rule.changed` events after commits so the data-plane can refresh.

## Rule semantics (MVP)
Exact-match fields:
- severity (enum)
- source (string)
- name (string)
Actions:
- notify via email endpoint (initially store email on rule or per-client)

## Required queries
- get all rules
- get rules by client_id
- get rules updated since timestamp
