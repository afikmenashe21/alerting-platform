output "cluster_id" {
  description = "ECS cluster ID"
  value       = aws_ecs_cluster.main.id
}

output "cluster_name" {
  description = "ECS cluster name"
  value       = aws_ecs_cluster.main.name
}

output "cluster_arn" {
  description = "ECS cluster ARN"
  value       = aws_ecs_cluster.main.arn
}

output "ecs_security_group_id" {
  description = "Security group ID for ECS instances"
  value       = aws_security_group.ecs_instances.id
}

output "alb_security_group_id" {
  description = "Security group ID for ALB (disabled)"
  value       = null  # No ALB security group
}

output "ecs_instance_role_arn" {
  description = "IAM role ARN for ECS instances"
  value       = aws_iam_role.ecs_instance.arn
}
