output "kafka_endpoint" {
  description = "Kafka bootstrap servers - using private IP since DNS resolution has issues with host network mode"
  # NOTE: This IP needs to be updated if the Kafka instance is replaced
  # TODO: Consider using an NLB or ECS service connect for dynamic resolution
  value       = "10.0.1.109:9092"
}

output "zookeeper_endpoint" {
  description = "Zookeeper endpoint"
  value       = "zookeeper.${aws_service_discovery_private_dns_namespace.main.name}:2181"
}

output "service_discovery_namespace" {
  description = "Service discovery namespace ID"
  value       = aws_service_discovery_private_dns_namespace.main.id
}
