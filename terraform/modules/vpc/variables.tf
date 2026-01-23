variable "project_name" {
  description = "Project name"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
}

variable "availability_zones" {
  description = "List of availability zones"
  type        = list(string)
}

variable "create_nat_gateway" {
  description = "Create NAT gateway for private subnets (costs ~$32/month, disable to save costs if services don't need outbound internet)"
  type        = bool
  default     = false # Changed to false to save $32/month
}
