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
  description = "Private subnet IDs for ECS instances"
  type        = list(string)
}

variable "ecs_instance_type" {
  description = "EC2 instance type for ECS cluster"
  type        = string
  default     = "t3.micro"
}

variable "desired_capacity" {
  description = "Desired number of ECS instances"
  type        = number
  default     = 2
}

variable "min_size" {
  description = "Minimum number of ECS instances"
  type        = number
  default     = 1
}

variable "max_size" {
  description = "Maximum number of ECS instances"
  type        = number
  default     = 4
}

variable "enable_container_insights" {
  description = "Enable CloudWatch Container Insights"
  type        = bool
  default     = false
}
