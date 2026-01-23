output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "ecs_cluster_name" {
  description = "ECS cluster name"
  value       = module.ecs_cluster.cluster_name
}

output "ecs_security_group_id" {
  description = "ECS instances security group ID"
  value       = module.ecs_cluster.ecs_security_group_id
}

output "private_subnet_ids" {
  description = "Private subnet IDs"
  value       = module.vpc.private_subnet_ids
}

output "rds_endpoint" {
  description = "RDS endpoint"
  value       = module.rds.endpoint
  sensitive   = true
}

output "rds_security_group_id" {
  description = "RDS security group ID"
  value       = module.rds.security_group_id
}

output "redis_endpoint" {
  description = "Redis endpoint"
  value       = module.redis.endpoint
}

output "kafka_endpoint" {
  description = "Kafka bootstrap servers"
  value       = module.kafka.kafka_endpoint
}

# Stable Elastic IP for API access
output "api_elastic_ip" {
  description = "Elastic IP for API access (stable, never changes)"
  value       = module.ecs_cluster.elastic_ip
}

output "api_endpoints" {
  description = "API endpoints for direct access"
  value = {
    rule_service    = "http://${module.ecs_cluster.elastic_ip}:8081"
    alert_producer  = "http://${module.ecs_cluster.elastic_ip}:8082"
  }
}

output "ecr_repository_urls" {
  description = "ECR repository URLs for all services"
  value       = module.ecr.repository_urls
}

output "deployment_commands" {
  description = "Commands to deploy services"
  value       = <<-EOT
    # 1. Authenticate Docker to ECR
    aws ecr get-login-password --region ${var.aws_region} | docker login --username AWS --password-stdin ${split("/", module.ecr.repository_urls["rule-service"])[0]}
    
    # 2. Build and push images (example for rule-service)
    docker build -f services/rule-service/Dockerfile -t ${module.ecr.repository_urls["rule-service"]}:latest .
    docker push ${module.ecr.repository_urls["rule-service"]}:latest
    
    # 3. Update ECS services to use new images
    aws ecs update-service --cluster ${module.ecs_cluster.cluster_name} --service rule-service --force-new-deployment --region ${var.aws_region}
    
    # See deployment guide for complete instructions
  EOT
}

# UI deployment commands disabled until CloudFront is available
