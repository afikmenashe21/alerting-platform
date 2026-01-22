# Core Configuration
variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
  default     = "alerting-platform"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "prod"
}

variable "aws_region" {
  description = "AWS region for deployment"
  type        = string
  default     = "us-east-1" # Free tier eligible
}

# Network Configuration
variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "Availability zones for the region"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}

# ECS Configuration
variable "ecs_instance_type" {
  description = "EC2 instance type for ECS cluster (t3.micro is free tier eligible)"
  type        = string
  default     = "t3.micro"
}

variable "ecs_desired_capacity" {
  description = "Desired number of EC2 instances in ECS cluster"
  type        = number
  default     = 2 # Minimum for high availability
}

variable "ecs_min_size" {
  description = "Minimum number of EC2 instances in ECS cluster"
  type        = number
  default     = 1
}

variable "ecs_max_size" {
  description = "Maximum number of EC2 instances in ECS cluster"
  type        = number
  default     = 4 # Can scale up as needed
}

# RDS Configuration
variable "db_instance_class" {
  description = "RDS instance class (db.t3.micro is free tier eligible)"
  type        = string
  default     = "db.t3.micro"
}

variable "db_name" {
  description = "Database name"
  type        = string
  default     = "alerting"
}

variable "db_username" {
  description = "Database master username"
  type        = string
  default     = "postgres"
  sensitive   = true
}

variable "db_password" {
  description = "Database master password"
  type        = string
  sensitive   = true
}

variable "db_allocated_storage" {
  description = "Allocated storage for RDS in GB (20GB is free tier eligible)"
  type        = number
  default     = 20
}

# Service Configuration
variable "service_desired_count" {
  description = "Default desired count for each service (1 for minimal cost, can scale up)"
  type        = number
  default     = 1
}

variable "service_max_count" {
  description = "Maximum count for auto-scaling services"
  type        = number
  default     = 3
}

# Container Configuration
variable "container_cpu" {
  description = "CPU units for containers (1024 = 1 vCPU)"
  type        = number
  default     = 256 # 0.25 vCPU - good for small services
}

variable "container_memory" {
  description = "Memory for containers in MB"
  type        = number
  default     = 512 # 512 MB
}

# Docker Image Configuration
variable "docker_registry" {
  description = "Docker registry URL (ECR will be created)"
  type        = string
  default     = "" # Will be populated with ECR URL
}

variable "image_tag" {
  description = "Docker image tag to deploy"
  type        = string
  default     = "latest"
}

# Kafka Configuration
variable "kafka_partitions" {
  description = "Number of partitions for Kafka topics"
  type        = number
  default     = 9
}

variable "kafka_replication_factor" {
  description = "Replication factor for Kafka topics (1 for single broker)"
  type        = number
  default     = 1
}

# Monitoring and Logging
variable "enable_container_insights" {
  description = "Enable CloudWatch Container Insights"
  type        = bool
  default     = false # Costs money, disable for free tier
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days"
  type        = number
  default     = 7 # Minimal retention for cost savings
}

# Tags
variable "tags" {
  description = "Common tags for all resources"
  type        = map(string)
  default = {
    Project     = "alerting-platform"
    ManagedBy   = "terraform"
    Environment = "prod"
  }
}
