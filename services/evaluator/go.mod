module evaluator

go 1.25.6

require (
	github.com/afikmenashe/alerting-platform/pkg/kafka v0.0.0
	github.com/afikmenashe/alerting-platform/pkg/proto v0.0.0
	github.com/redis/go-redis/v9 v9.5.1
	github.com/segmentio/kafka-go v0.4.47
	google.golang.org/protobuf v1.32.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
)

replace github.com/afikmenashe/alerting-platform/pkg/proto => ../../pkg/proto

replace github.com/afikmenashe/alerting-platform/pkg/kafka => ../../pkg/kafka
