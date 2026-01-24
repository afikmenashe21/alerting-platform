variable "project_name" {
  description = "Project name"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "service_name" {
  description = "Service name"
  type        = string
}

variable "ecs_cluster_id" {
  description = "ECS cluster ID"
  type        = string
}

variable "ecs_cluster_name" {
  description = "ECS cluster name"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs"
  type        = list(string)
}

variable "ecs_security_group_id" {
  description = "ECS security group ID"
  type        = string
}

variable "container_image" {
  description = "Docker image for the container"
  type        = string
}

variable "container_port" {
  description = "Port exposed by the container (0 for no port)"
  type        = number
  default     = 0
}

variable "container_cpu" {
  description = "CPU units for the container"
  type        = number
  default     = 256
}

variable "container_memory" {
  description = "Memory for the container in MB"
  type        = number
  default     = 512
}

variable "desired_count" {
  description = "Desired number of tasks"
  type        = number
  default     = 1
}

variable "max_count" {
  description = "Maximum number of tasks for auto-scaling"
  type        = number
  default     = 3
}

variable "environment_variables" {
  description = "Environment variables for the container"
  type        = map(string)
  default     = {}
}

variable "load_balancer_enabled" {
  description = "Whether to attach a load balancer"
  type        = bool
  default     = false
}

variable "target_group_arn" {
  description = "Target group ARN for load balancer"
  type        = string
  default     = ""
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days"
  type        = number
  default     = 7
}

variable "use_host_network" {
  description = "Use host network mode instead of bridge (for service discovery)"
  type        = bool
  default     = false
}

variable "service_discovery_namespace_id" {
  description = "Service discovery namespace ID (required if use_host_network is true)"
  type        = string
  default     = ""
}

variable "enable_service_discovery" {
  description = "Register service with Cloud Map"
  type        = bool
  default     = false
}

variable "task_role_policy_json" {
  description = "Optional inline IAM policy JSON for the task role"
  type        = string
  default     = ""
}

variable "secrets" {
  description = "Secrets to inject from SSM Parameter Store (map of ENV_VAR_NAME => SSM parameter ARN)"
  type        = map(string)
  default     = {}
}
