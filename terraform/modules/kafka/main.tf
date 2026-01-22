# Since AWS MSK is expensive, we'll run Kafka as ECS services
# This provides a cost-effective solution for dev/staging environments

# CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "zookeeper" {
  name              = "/ecs/${var.project_name}/${var.environment}/zookeeper"
  retention_in_days = var.log_retention_days

  tags = {
    Name = "${var.project_name}-${var.environment}-zookeeper-logs"
  }
}

resource "aws_cloudwatch_log_group" "kafka" {
  name              = "/ecs/${var.project_name}/${var.environment}/kafka"
  retention_in_days = var.log_retention_days

  tags = {
    Name = "${var.project_name}-${var.environment}-kafka-logs"
  }
}

# IAM Role for ECS Task Execution
resource "aws_iam_role" "kafka_task_execution" {
  name = "${var.project_name}-${var.environment}-kafka-task-execution"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "kafka_task_execution" {
  role       = aws_iam_role.kafka_task_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# EFS for Kafka data persistence (optional, costs money)
# Commented out for free tier, but can be enabled for production
# resource "aws_efs_file_system" "kafka" {
#   creation_token = "${var.project_name}-${var.environment}-kafka-data"
#   encrypted      = true
#   tags = {
#     Name = "${var.project_name}-${var.environment}-kafka-data"
#   }
# }

# Zookeeper Task Definition
# Using host network mode for simple DNS resolution
# Container binds directly to host's network interface
# Memory optimized for low-cost deployment on t3.small
resource "aws_ecs_task_definition" "zookeeper" {
  family                   = "${var.project_name}-${var.environment}-zookeeper"
  network_mode             = "host"  # Host mode for direct network access
  requires_compatibilities = ["EC2"]
  cpu                      = "128"
  memory                   = "192"
  execution_role_arn       = aws_iam_role.kafka_task_execution.arn

  container_definitions = jsonencode([
    {
      name      = "zookeeper"
      image     = var.zookeeper_image
      essential = true

      portMappings = [
        {
          containerPort = 2181
          hostPort      = 2181
          protocol      = "tcp"
        }
      ]

      environment = [
        {
          name  = "ZOOKEEPER_CLIENT_PORT"
          value = "2181"
        },
        {
          name  = "ZOOKEEPER_TICK_TIME"
          value = "2000"
        },
        {
          name  = "KAFKA_HEAP_OPTS"
          value = "-Xmx128M -Xms64M"
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.zookeeper.name
          "awslogs-region"        = data.aws_region.current.name
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])

  tags = {
    Name = "${var.project_name}-${var.environment}-zookeeper"
  }
}

# Kafka Task Definition
# Using host network mode for simple DNS resolution
# Container binds directly to host's network interface
# Memory optimized for low-cost deployment on t3.small
resource "aws_ecs_task_definition" "kafka" {
  family                   = "${var.project_name}-${var.environment}-kafka"
  network_mode             = "host"  # Host mode for direct network access
  requires_compatibilities = ["EC2"]
  cpu                      = "256"
  memory                   = "384"
  execution_role_arn       = aws_iam_role.kafka_task_execution.arn

  container_definitions = jsonencode([
    {
      name      = "kafka"
      image     = var.kafka_image
      essential = true

      portMappings = [
        {
          containerPort = 9092
          hostPort      = 9092
          protocol      = "tcp"
        },
        {
          containerPort = 29092
          hostPort      = 29092
          protocol      = "tcp"
        }
      ]

      environment = [
        {
          name  = "KAFKA_BROKER_ID"
          value = "1"
        },
        {
          name  = "KAFKA_ZOOKEEPER_CONNECT"
          value = "localhost:2181"
        },
        {
          name  = "KAFKA_LISTENER_SECURITY_PROTOCOL_MAP"
          value = "PLAINTEXT:PLAINTEXT"
        },
        {
          name  = "KAFKA_LISTENERS"
          value = "PLAINTEXT://0.0.0.0:9092"
        },
        {
          name  = "KAFKA_ADVERTISED_LISTENERS"
          value = "PLAINTEXT://10.0.1.109:9092"
        },
        {
          name  = "KAFKA_INTER_BROKER_LISTENER_NAME"
          value = "PLAINTEXT"
        },
        {
          name  = "KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR"
          value = "1"
        },
        {
          name  = "KAFKA_TRANSACTION_STATE_LOG_MIN_ISR"
          value = "1"
        },
        {
          name  = "KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR"
          value = "1"
        },
        {
          name  = "KAFKA_AUTO_CREATE_TOPICS_ENABLE"
          value = "true"
        },
        {
          name  = "KAFKA_HEAP_OPTS"
          value = "-Xmx256M -Xms128M"
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.kafka.name
          "awslogs-region"        = data.aws_region.current.name
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])

  tags = {
    Name = "${var.project_name}-${var.environment}-kafka"
  }
}

# Service Discovery Namespace
resource "aws_service_discovery_private_dns_namespace" "main" {
  name        = "${var.project_name}-${var.environment}.local"
  description = "Private DNS namespace for service discovery"
  vpc         = var.vpc_id

  tags = {
    Name = "${var.project_name}-${var.environment}-service-discovery"
  }
}

# Service Discovery for Kafka - using A record for simple DNS resolution
resource "aws_service_discovery_service" "kafka" {
  name = "kafka"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.main.id

    dns_records {
      ttl  = 10
      type = "A"
    }

    routing_policy = "MULTIVALUE"
  }

  health_check_custom_config {
    failure_threshold = 1
  }
}

# Zookeeper ECS Service
# Using host network mode - no service discovery (Kafka connects via localhost)
resource "aws_ecs_service" "zookeeper" {
  name            = "zookeeper"
  cluster         = var.ecs_cluster_id
  task_definition = aws_ecs_task_definition.zookeeper.arn
  desired_count   = 1
  launch_type     = "EC2"

  # No service discovery - Kafka connects to Zookeeper via localhost

  tags = {
    Name = "${var.project_name}-${var.environment}-zookeeper"
  }
}

# Kafka ECS Service
# Using host network mode - no service discovery (services connect via private IP)
resource "aws_ecs_service" "kafka" {
  name            = "kafka"
  cluster         = var.ecs_cluster_id
  task_definition = aws_ecs_task_definition.kafka.arn
  desired_count   = 1
  launch_type     = "EC2"

  # No service discovery - services will connect to Kafka via the instance private IP
  # The Kafka advertised listeners will use the instance metadata to get the IP

  depends_on = [aws_ecs_service.zookeeper]

  tags = {
    Name = "${var.project_name}-${var.environment}-kafka"
  }
}

data "aws_region" "current" {}
