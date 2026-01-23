terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Uncomment for remote state management
  # backend "s3" {
  #   bucket         = "alerting-platform-terraform-state"
  #   key            = "prod/terraform.tfstate"
  #   region         = "us-east-1"
  #   dynamodb_table = "alerting-platform-terraform-locks"
  #   encrypt        = true
  # }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = var.tags
  }
}

# VPC and Networking
module "vpc" {
  source = "./modules/vpc"

  project_name       = var.project_name
  environment        = var.environment
  vpc_cidr           = var.vpc_cidr
  availability_zones = var.availability_zones
}

# ECR Repositories for Docker Images
module "ecr" {
  source = "./modules/ecr"

  project_name = var.project_name
  services = [
    "rule-service",
    "rule-updater",
    "evaluator",
    "aggregator",
    "sender",
    "alert-producer"
  ]
}

# ECS Cluster
# Note: Using public subnets for ECS instances to avoid NAT Gateway costs (~$32/month)
# This is safe because security groups block all inbound traffic except from within the SG
module "ecs_cluster" {
  source = "./modules/ecs-cluster"

  project_name              = var.project_name
  environment               = var.environment
  vpc_id                    = module.vpc.vpc_id
  private_subnet_ids        = module.vpc.public_subnet_ids # Using public subnets to avoid NAT Gateway costs
  ecs_instance_type         = var.ecs_instance_type
  desired_capacity          = var.ecs_desired_capacity
  min_size                  = var.ecs_min_size
  max_size                  = var.ecs_max_size
  enable_container_insights = var.enable_container_insights
}

# RDS Postgres
module "rds" {
  source = "./modules/rds"

  project_name          = var.project_name
  environment           = var.environment
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  db_name               = var.db_name
  db_username           = var.db_username
  db_password           = var.db_password
  db_instance_class     = var.db_instance_class
  allocated_storage     = var.db_allocated_storage
  ecs_security_group_id = module.ecs_cluster.ecs_security_group_id
}

# ElastiCache Redis
module "redis" {
  source = "./modules/redis"

  project_name          = var.project_name
  environment           = var.environment
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  ecs_security_group_id = module.ecs_cluster.ecs_security_group_id
}

# Kafka on ECS (since MSK is not free tier)
module "kafka" {
  source = "./modules/kafka"

  project_name          = var.project_name
  environment           = var.environment
  ecs_cluster_id        = module.ecs_cluster.cluster_id
  ecs_cluster_name      = module.ecs_cluster.cluster_name
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  ecs_security_group_id = module.ecs_cluster.ecs_security_group_id
  log_retention_days    = var.log_retention_days
  kafka_image           = "confluentinc/cp-kafka:7.5.0"
  zookeeper_image       = "confluentinc/cp-zookeeper:7.5.0"
}

# Application Load Balancer for rule-service API
# TEMPORARILY DISABLED - AWS account doesn't support ALB yet
# Contact AWS Support to enable, then uncomment this
# module "alb" {
#   source = "./modules/alb"
#
#   project_name       = var.project_name
#   environment        = var.environment
#   vpc_id             = module.vpc.vpc_id
#   public_subnet_ids  = module.vpc.public_subnet_ids
# }

# ECS Services
module "rule_service" {
  source = "./modules/ecs-service"

  project_name          = var.project_name
  environment           = var.environment
  service_name          = "rule-service"
  ecs_cluster_id        = module.ecs_cluster.cluster_id
  ecs_cluster_name      = module.ecs_cluster.cluster_name
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  ecs_security_group_id = module.ecs_cluster.ecs_security_group_id

  container_image  = "${module.ecr.repository_urls["rule-service"]}:${var.image_tag}"
  container_port   = 8081 # rule-service listens on 8081
  container_cpu    = var.container_cpu
  container_memory = var.container_memory

  desired_count = var.service_desired_count
  max_count     = var.service_max_count

  # Host network mode for direct public access (no Lambda/API Gateway needed)
  use_host_network         = true
  enable_service_discovery = false

  environment_variables = {
    HTTP_PORT          = "8081" # Explicitly set to match service code default
    KAFKA_BROKERS      = module.kafka.kafka_endpoint
    POSTGRES_DSN       = "postgres://${var.db_username}:${var.db_password}@${module.rds.endpoint}/${var.db_name}?sslmode=require"
    RULE_CHANGED_TOPIC = "rule.changed"
  }

  load_balancer_enabled = false
  log_retention_days    = var.log_retention_days
}

module "rule_updater" {
  source = "./modules/ecs-service"

  project_name          = var.project_name
  environment           = var.environment
  service_name          = "rule-updater"
  ecs_cluster_id        = module.ecs_cluster.cluster_id
  ecs_cluster_name      = module.ecs_cluster.cluster_name
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  ecs_security_group_id = module.ecs_cluster.ecs_security_group_id

  container_image  = "${module.ecr.repository_urls["rule-updater"]}:${var.image_tag}"
  container_port   = 0 # No external port
  container_cpu    = var.container_cpu
  container_memory = var.container_memory

  desired_count = 1 # MUST BE 1 - writes Redis snapshot
  max_count     = 1 # NEVER scale this service

  environment_variables = {
    KAFKA_BROKERS      = module.kafka.kafka_endpoint
    POSTGRES_DSN       = "postgres://${var.db_username}:${var.db_password}@${module.rds.endpoint}/${var.db_name}?sslmode=require"
    REDIS_ADDR         = module.redis.endpoint
    RULE_CHANGED_TOPIC = "rule.changed"
    CONSUMER_GROUP_ID  = "rule-updater-group"
  }

  load_balancer_enabled = false
  log_retention_days    = var.log_retention_days
}

module "evaluator" {
  source = "./modules/ecs-service"

  project_name          = var.project_name
  environment           = var.environment
  service_name          = "evaluator"
  ecs_cluster_id        = module.ecs_cluster.cluster_id
  ecs_cluster_name      = module.ecs_cluster.cluster_name
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  ecs_security_group_id = module.ecs_cluster.ecs_security_group_id

  container_image  = "${module.ecr.repository_urls["evaluator"]}:${var.image_tag}"
  container_port   = 0 # No external port
  container_cpu    = var.container_cpu
  container_memory = var.container_memory

  desired_count = var.service_desired_count
  max_count     = var.service_max_count

  environment_variables = {
    KAFKA_BROKERS         = module.kafka.kafka_endpoint
    REDIS_ADDR            = module.redis.endpoint
    ALERTS_NEW_TOPIC      = "alerts.new"
    ALERTS_MATCHED_TOPIC  = "alerts.matched"
    RULE_CHANGED_TOPIC    = "rule.changed"
    CONSUMER_GROUP_ID     = "evaluator-group"
    RULE_CHANGED_GROUP_ID = "evaluator-rule-changed-group"
  }

  load_balancer_enabled = false
  log_retention_days    = var.log_retention_days
}

module "aggregator" {
  source = "./modules/ecs-service"

  project_name          = var.project_name
  environment           = var.environment
  service_name          = "aggregator"
  ecs_cluster_id        = module.ecs_cluster.cluster_id
  ecs_cluster_name      = module.ecs_cluster.cluster_name
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  ecs_security_group_id = module.ecs_cluster.ecs_security_group_id

  container_image  = "${module.ecr.repository_urls["aggregator"]}:${var.image_tag}"
  container_port   = 0 # No external port
  container_cpu    = var.container_cpu
  container_memory = var.container_memory

  desired_count = var.service_desired_count
  max_count     = var.service_max_count

  environment_variables = {
    KAFKA_BROKERS             = module.kafka.kafka_endpoint
    POSTGRES_DSN              = "postgres://${var.db_username}:${var.db_password}@${module.rds.endpoint}/${var.db_name}?sslmode=require"
    ALERTS_MATCHED_TOPIC      = "alerts.matched"
    NOTIFICATIONS_READY_TOPIC = "notifications.ready"
    CONSUMER_GROUP_ID         = "aggregator-group"
  }

  load_balancer_enabled = false
  log_retention_days    = var.log_retention_days
}

module "sender" {
  source = "./modules/ecs-service"

  project_name          = var.project_name
  environment           = var.environment
  service_name          = "sender"
  ecs_cluster_id        = module.ecs_cluster.cluster_id
  ecs_cluster_name      = module.ecs_cluster.cluster_name
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  ecs_security_group_id = module.ecs_cluster.ecs_security_group_id

  container_image  = "${module.ecr.repository_urls["sender"]}:${var.image_tag}"
  container_port   = 0 # No external port
  container_cpu    = var.container_cpu
  container_memory = var.container_memory

  desired_count = var.service_desired_count
  max_count     = var.service_max_count

  environment_variables = {
    KAFKA_BROKERS             = module.kafka.kafka_endpoint
    POSTGRES_DSN              = "postgres://${var.db_username}:${var.db_password}@${module.rds.endpoint}/${var.db_name}?sslmode=require"
    NOTIFICATIONS_READY_TOPIC = "notifications.ready"
    CONSUMER_GROUP_ID         = "sender-group"
    AWS_REGION                = var.aws_region
    SES_FROM                  = var.smtp_from
  }

  task_role_policy_json = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ses:SendEmail",
          "ses:SendRawEmail"
        ]
        Resource = "*"
      }
    ]
  })

  load_balancer_enabled = false
  log_retention_days    = var.log_retention_days
}

module "alert_producer" {
  source = "./modules/ecs-service"

  project_name          = var.project_name
  environment           = var.environment
  service_name          = "alert-producer"
  ecs_cluster_id        = module.ecs_cluster.cluster_id
  ecs_cluster_name      = module.ecs_cluster.cluster_name
  vpc_id                = module.vpc.vpc_id
  private_subnet_ids    = module.vpc.private_subnet_ids
  ecs_security_group_id = module.ecs_cluster.ecs_security_group_id

  container_image  = "${module.ecr.repository_urls["alert-producer"]}:${var.image_tag}"
  container_port   = 8082 # alert-producer-api listens on 8082
  container_cpu    = var.container_cpu
  container_memory = var.container_memory

  desired_count = var.service_desired_count
  max_count     = var.service_max_count

  # Host network mode for direct public access (no Lambda/API Gateway needed)
  use_host_network         = true
  enable_service_discovery = false

  environment_variables = {
    KAFKA_BROKERS    = module.kafka.kafka_endpoint
    ALERTS_NEW_TOPIC = "alerts.new"
  }

  load_balancer_enabled = false
  log_retention_days    = var.log_retention_days
}

# =============================================================================
# API Gateway HTTPS Proxy
# =============================================================================
# Provides HTTPS endpoint for GitHub Pages UI (mixed content fix)
# Proxies requests to HTTP backend on Elastic IP
module "api_gateway" {
  source = "./modules/api-gateway-proxy"

  project_name = var.project_name
  environment  = var.environment
  backend_ip   = module.ecs_cluster.elastic_ip
}
