variable "project_name" {
  description = "Project name"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "backend_ip" {
  description = "Backend EC2 public IP (Elastic IP)"
  type        = string
}
