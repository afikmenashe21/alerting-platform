module metrics-service

go 1.23

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/afikmenashe/alerting-platform/pkg/metrics v0.0.0
	github.com/afikmenashe/alerting-platform/pkg/shared v0.0.0
	github.com/lib/pq v1.10.9
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/redis/go-redis/v9 v9.7.3 // indirect
)

replace github.com/afikmenashe/alerting-platform/pkg/metrics => ../../pkg/metrics

replace github.com/afikmenashe/alerting-platform/pkg/shared => ../../pkg/shared
