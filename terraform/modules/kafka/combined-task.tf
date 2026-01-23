# Combined Kafka + Zookeeper Task Definition
# Both containers in a single task with awsvpc networking.
# Uses AWS Cloud Map service discovery for DNS-based connectivity:
#   kafka.alerting-platform-prod.local:9092
# This eliminates hardcoded IPs and handles instance mobility automatically.

resource "aws_ecs_task_definition" "kafka_combined" {
  family                   = "${var.project_name}-${var.environment}-kafka-combined"
  network_mode             = "awsvpc"
  requires_compatibilities = ["EC2"]
  cpu                      = "512"
  memory                   = "640"
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
        { name = "ZOOKEEPER_CLIENT_PORT", value = "2181" },
        { name = "ZOOKEEPER_TICK_TIME", value = "2000" },
        { name = "KAFKA_HEAP_OPTS", value = "-Xmx128M -Xms64M" }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.zookeeper.name
          "awslogs-region"        = data.aws_region.current.name
          "awslogs-stream-prefix" = "ecs"
        }
      }
    },
    {
      name      = "kafka"
      image     = var.kafka_image
      essential = true

      dependsOn = [
        { containerName = "zookeeper", condition = "START" }
      ]

      portMappings = [
        {
          containerPort = 9092
          hostPort      = 9092
          protocol      = "tcp"
        }
      ]

      environment = [
        { name = "KAFKA_BROKER_ID", value = "1" },
        { name = "KAFKA_ZOOKEEPER_CONNECT", value = "localhost:2181" },
        { name = "KAFKA_LISTENER_SECURITY_PROTOCOL_MAP", value = "PLAINTEXT:PLAINTEXT" },
        { name = "KAFKA_LISTENERS", value = "PLAINTEXT://0.0.0.0:9092" },
        { name = "KAFKA_ADVERTISED_LISTENERS", value = "PLAINTEXT://kafka.${var.project_name}-${var.environment}.local:9092" },
        { name = "KAFKA_INTER_BROKER_LISTENER_NAME", value = "PLAINTEXT" },
        { name = "KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR", value = "1" },
        { name = "KAFKA_TRANSACTION_STATE_LOG_MIN_ISR", value = "1" },
        { name = "KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR", value = "1" },
        { name = "KAFKA_AUTO_CREATE_TOPICS_ENABLE", value = "true" },
        { name = "KAFKA_HEAP_OPTS", value = "-Xmx256M -Xms128M" }
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
    Name = "${var.project_name}-${var.environment}-kafka-combined"
  }
}

# ECS Service for combined Kafka + Zookeeper with service discovery
resource "aws_ecs_service" "kafka_combined" {
  name            = "kafka-combined"
  cluster         = var.ecs_cluster_id
  task_definition = aws_ecs_task_definition.kafka_combined.arn
  desired_count   = 1
  launch_type     = "EC2"

  network_configuration {
    subnets         = var.private_subnet_ids
    security_groups = [var.ecs_security_group_id]
  }

  service_registries {
    registry_arn   = aws_service_discovery_service.kafka.arn
    container_name = "kafka"
    container_port = 9092
  }

  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent         = 100

  tags = {
    Name = "${var.project_name}-${var.environment}-kafka-combined"
  }
}

# Service Discovery for Kafka DNS resolution
resource "aws_service_discovery_service" "kafka" {
  name = "kafka"

  dns_config {
    namespace_id = var.service_discovery_namespace_id

    dns_records {
      type = "A"
      ttl  = 10
    }

    routing_policy = "MULTIVALUE"
  }

  health_check_custom_config {
    failure_threshold = 1
  }
}
