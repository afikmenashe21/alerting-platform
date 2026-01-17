# evaluator – Tech Context

- Go 1.22+
- Kafka consumer group for `alerts.new`, producer for `alerts.matched` (kafka-go)
- Redis (go-redis): read `rules:snapshot`, poll `rules:version`
- No Postgres dependency in evaluator (rules come from Redis)
- JSON contracts with schema_version
