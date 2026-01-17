# evaluator – System Patterns

## In-memory indexes
We avoid a giant map of all combinations because rule space can explode with wildcards later.

For exact match MVP:
- bySeverity[severity] -> []ruleInt
- bySource[source] -> []ruleInt
- byName[name] -> []ruleInt
- ruleInt -> {rule_id, client_id} (store ids once)

### Matching algorithm (intersection)
1. candidates = smallest of the three lists
2. use boolean mark/set for intersection with other two lists
3. output list of matched ruleInt
4. group by client_id

## Refresh strategy
- Poll Redis `rules:version` every N seconds.
- If changed:
  - reload `rules:snapshot`
  - rebuild indexes atomically (swap pointer)

This makes evaluator restart fast and handles rule updates without Kafka replays.
