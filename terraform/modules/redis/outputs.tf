output "endpoint" {
  description = "Redis endpoint (address:port)"
  value       = "${aws_elasticache_cluster.redis.cache_nodes[0].address}:${aws_elasticache_cluster.redis.cache_nodes[0].port}"
}

output "address" {
  description = "Redis address (hostname)"
  value       = aws_elasticache_cluster.redis.cache_nodes[0].address
}

output "port" {
  description = "Redis port"
  value       = aws_elasticache_cluster.redis.cache_nodes[0].port
}

output "security_group_id" {
  description = "Security group ID for Redis"
  value       = aws_security_group.redis.id
}
