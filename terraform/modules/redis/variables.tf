variable "project_name" {
  description = "Project name"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for Redis"
  type        = list(string)
}

variable "node_type" {
  description = "ElastiCache node type (cache.t3.micro is free tier eligible)"
  type        = string
  default     = "cache.t3.micro"
}

variable "ecs_security_group_id" {
  description = "Security group ID of ECS instances"
  type        = string
}

variable "snapshot_retention_limit" {
  description = "Number of days to retain snapshots"
  type        = number
  default     = 5
}
