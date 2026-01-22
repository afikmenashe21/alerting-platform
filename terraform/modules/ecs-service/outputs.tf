output "service_name" {
  description = "ECS service name"
  value       = aws_ecs_service.service.name
}

output "service_arn" {
  description = "ECS service ARN"
  value       = aws_ecs_service.service.id
}

output "task_definition_arn" {
  description = "Task definition ARN"
  value       = aws_ecs_task_definition.service.arn
}

output "task_execution_role_arn" {
  description = "Task execution role ARN"
  value       = aws_iam_role.task_execution.arn
}

output "task_role_arn" {
  description = "Task role ARN"
  value       = aws_iam_role.task_role.arn
}

output "log_group_name" {
  description = "CloudWatch log group name"
  value       = aws_cloudwatch_log_group.service.name
}
