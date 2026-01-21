module rule-service

go 1.25.6

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/afikmenashe/alerting-platform/pkg/kafka v0.0.0
	github.com/afikmenashe/alerting-platform/pkg/proto v0.0.0
	github.com/lib/pq v1.10.9
	github.com/segmentio/kafka-go v0.4.47
	google.golang.org/protobuf v1.32.0
)

require (
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
)

replace github.com/afikmenashe/alerting-platform/pkg/proto => ../../pkg/proto

replace github.com/afikmenashe/alerting-platform/pkg/kafka => ../../pkg/kafka
