# CloudWatch Log Group
resource "aws_cloudwatch_log_group" "service" {
  name              = "/ecs/${var.project_name}/${var.environment}/${var.service_name}"
  retention_in_days = var.log_retention_days

  tags = {
    Name    = "${var.project_name}-${var.environment}-${var.service_name}-logs"
    Service = var.service_name
  }
}

# IAM Role for ECS Task Execution
resource "aws_iam_role" "task_execution" {
  name = "${var.project_name}-${var.environment}-${var.service_name}-execution"

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

  tags = {
    Name    = "${var.project_name}-${var.environment}-${var.service_name}-execution-role"
    Service = var.service_name
  }
}

resource "aws_iam_role_policy_attachment" "task_execution" {
  role       = aws_iam_role.task_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# Allow task execution role to read SSM parameters (for secrets)
resource "aws_iam_role_policy" "task_execution_ssm" {
  count = length(var.secrets) > 0 ? 1 : 0
  name  = "${var.project_name}-${var.environment}-${var.service_name}-ssm-read"
  role  = aws_iam_role.task_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameters",
          "ssm:GetParameter"
        ]
        Resource = values(var.secrets)
      }
    ]
  })
}

# IAM Role for ECS Task (application permissions)
resource "aws_iam_role" "task_role" {
  name = "${var.project_name}-${var.environment}-${var.service_name}-task"

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

  tags = {
    Name    = "${var.project_name}-${var.environment}-${var.service_name}-task-role"
    Service = var.service_name
  }
}

# Optional inline policy for the task role (e.g., SES permissions for sender)
resource "aws_iam_role_policy" "task_role_policy" {
  count  = var.task_role_policy_json != "" ? 1 : 0
  name   = "${var.project_name}-${var.environment}-${var.service_name}-task-policy"
  role   = aws_iam_role.task_role.id
  policy = var.task_role_policy_json
}

# Convert environment variables map to list format for ECS
locals {
  environment_list = [
    for key, value in var.environment_variables : {
      name  = key
      value = value
    }
  ]

  # Convert secrets map to list format for ECS (SSM Parameter Store references)
  secrets_list = [
    for key, value in var.secrets : {
      name      = key
      valueFrom = value
    }
  ]
}

# ECS Task Definition
# Supports both bridge and host network modes
resource "aws_ecs_task_definition" "service" {
  family                   = "${var.project_name}-${var.environment}-${var.service_name}"
  network_mode             = var.use_host_network ? "host" : "bridge"
  requires_compatibilities = ["EC2"]
  cpu                      = var.container_cpu
  memory                   = var.container_memory
  execution_role_arn       = aws_iam_role.task_execution.arn
  task_role_arn            = aws_iam_role.task_role.arn

  container_definitions = jsonencode([
    {
      name      = var.service_name
      image     = var.container_image
      essential = true

      # For host mode: fixed port, for bridge mode: dynamic port (0)
      portMappings = var.container_port > 0 ? [
        {
          containerPort = var.container_port
          hostPort      = var.use_host_network ? var.container_port : 0
          protocol      = "tcp"
        }
      ] : []

      environment = local.environment_list
      secrets     = length(local.secrets_list) > 0 ? local.secrets_list : null

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.service.name
          "awslogs-region"        = data.aws_region.current.name
          "awslogs-stream-prefix" = "ecs"
        }
      }

      # Health checks disabled - Docker images don't have wget installed
      # and we're not using ALB, so ECS health checks aren't needed
      # TODO: Re-enable with curl after adding it to Dockerfiles
      healthCheck = null
    }
  ])

  tags = {
    Name    = "${var.project_name}-${var.environment}-${var.service_name}"
    Service = var.service_name
  }
}

# Service Discovery Service (for host network mode)
resource "aws_service_discovery_service" "service" {
  count = var.enable_service_discovery ? 1 : 0
  name  = var.service_name

  dns_config {
    namespace_id = var.service_discovery_namespace_id

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

# ECS Service
# Note: No network_configuration needed for bridge/host network mode
resource "aws_ecs_service" "service" {
  name            = var.service_name
  cluster         = var.ecs_cluster_id
  task_definition = aws_ecs_task_definition.service.arn
  desired_count   = var.desired_count
  launch_type     = "EC2"

  # Host network mode: stop old task before starting new (can't share ports)
  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent         = 100

  # network_configuration not needed for bridge/host mode - removed to fix deployment

  dynamic "load_balancer" {
    for_each = var.load_balancer_enabled ? [1] : []
    content {
      target_group_arn = var.target_group_arn
      container_name   = var.service_name
      container_port   = var.container_port
    }
  }

  # Service discovery registration (only for host network mode)
  dynamic "service_registries" {
    for_each = var.enable_service_discovery ? [1] : []
    content {
      registry_arn = aws_service_discovery_service.service[0].arn
    }
  }

  tags = {
    Name    = "${var.project_name}-${var.environment}-${var.service_name}"
    Service = var.service_name
  }

  # Ignore changes to desired_count when using auto-scaling
  lifecycle {
    ignore_changes = [desired_count]
  }
}

# Auto Scaling Target
resource "aws_appautoscaling_target" "service" {
  count              = var.max_count > var.desired_count ? 1 : 0
  max_capacity       = var.max_count
  min_capacity       = var.desired_count
  resource_id        = "service/${var.ecs_cluster_name}/${aws_ecs_service.service.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

# Auto Scaling Policy - CPU
resource "aws_appautoscaling_policy" "cpu" {
  count              = var.max_count > var.desired_count ? 1 : 0
  name               = "${var.project_name}-${var.environment}-${var.service_name}-cpu"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.service[0].resource_id
  scalable_dimension = aws_appautoscaling_target.service[0].scalable_dimension
  service_namespace  = aws_appautoscaling_target.service[0].service_namespace

  target_tracking_scaling_policy_configuration {
    target_value       = 70.0
    scale_in_cooldown  = 300
    scale_out_cooldown = 60

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}

# Auto Scaling Policy - Memory
resource "aws_appautoscaling_policy" "memory" {
  count              = var.max_count > var.desired_count ? 1 : 0
  name               = "${var.project_name}-${var.environment}-${var.service_name}-memory"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.service[0].resource_id
  scalable_dimension = aws_appautoscaling_target.service[0].scalable_dimension
  service_namespace  = aws_appautoscaling_target.service[0].service_namespace

  target_tracking_scaling_policy_configuration {
    target_value       = 80.0
    scale_in_cooldown  = 300
    scale_out_cooldown = 60

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageMemoryUtilization"
    }
  }
}

data "aws_region" "current" {}
