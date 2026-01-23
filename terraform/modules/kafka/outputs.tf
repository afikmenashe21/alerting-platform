output "kafka_endpoint" {
  description = "Kafka bootstrap servers via DNS service discovery"
  value       = "kafka.${aws_service_discovery_private_dns_namespace.main.name}:9092"
}

output "zookeeper_endpoint" {
  description = "Zookeeper endpoint"
  value       = "zookeeper.${aws_service_discovery_private_dns_namespace.main.name}:2181"
}

output "service_discovery_namespace_id" {
  description = "Service discovery namespace ID"
  value       = aws_service_discovery_private_dns_namespace.main.id
}

output "service_discovery_namespace_name" {
  description = "Service discovery namespace name"
  value       = aws_service_discovery_private_dns_namespace.main.name
}
