# sender – Project Brief

## Purpose
Consume `notifications.ready` and perform the side-effect: send email (stub) and mark notification status.

## Input
`notifications.ready`: {notification_id, client_id, alert_id}

## Data
Reads `notifications` row from Postgres to get payload and matched_rule_ids.

## Output
Updates notification status to SENT/FAILED.
