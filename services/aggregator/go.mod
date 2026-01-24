module aggregator

go 1.23

require (
	github.com/afikmenashe/alerting-platform/pkg/kafka v0.0.0
	github.com/afikmenashe/alerting-platform/pkg/metrics v0.0.0
	github.com/lib/pq v1.10.9
	github.com/redis/go-redis/v9 v9.7.3
	github.com/segmentio/kafka-go v0.4.47
)

require (
	github.com/afikmenashe/alerting-platform/pkg/proto v0.0.0
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	google.golang.org/protobuf v1.32.0
)

replace github.com/afikmenashe/alerting-platform/pkg/proto => ../../pkg/proto

replace github.com/afikmenashe/alerting-platform/pkg/kafka => ../../pkg/kafka

replace github.com/afikmenashe/alerting-platform/pkg/metrics => ../../pkg/metrics
