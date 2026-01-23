# Simple API Gateway HTTP Proxy
# Provides HTTPS endpoint that proxies to HTTP backend
# No Lambda needed - direct HTTP integration

# API Gateway HTTP API
resource "aws_apigatewayv2_api" "main" {
  name          = "${var.project_name}-${var.environment}-api"
  protocol_type = "HTTP"

  cors_configuration {
    allow_origins = ["*"]
    allow_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_headers = ["Content-Type", "Authorization"]
    max_age       = 3600
  }

  tags = {
    Name = "${var.project_name}-${var.environment}-api"
  }
}

# Integration for rule-service
# Route captures /api/{proxy+}, integration adds /api/ prefix
resource "aws_apigatewayv2_integration" "rule_service" {
  api_id             = aws_apigatewayv2_api.main.id
  integration_type   = "HTTP_PROXY"
  integration_method = "ANY"
  integration_uri    = "http://${var.backend_ip}:8081/api/{proxy}"
}

# Integration for alert-producer
# Route captures /alert-producer-api/{proxy+}, integration forwards as /{proxy}
resource "aws_apigatewayv2_integration" "alert_producer" {
  api_id             = aws_apigatewayv2_api.main.id
  integration_type   = "HTTP_PROXY"
  integration_method = "ANY"
  integration_uri    = "http://${var.backend_ip}:8082/{proxy}"
}

# Route: /api/* -> rule-service
# Captures: /api/v1/clients -> proxy=v1/clients -> backend gets /api/v1/clients
resource "aws_apigatewayv2_route" "rule_service" {
  api_id    = aws_apigatewayv2_api.main.id
  route_key = "ANY /api/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.rule_service.id}"
}

# Route: /alert-producer-api/* -> alert-producer  
# Captures: /alert-producer-api/api/v1/alerts -> proxy=api/v1/alerts -> backend gets /api/v1/alerts
resource "aws_apigatewayv2_route" "alert_producer" {
  api_id    = aws_apigatewayv2_api.main.id
  route_key = "ANY /alert-producer-api/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.alert_producer.id}"
}

# Default stage with auto-deploy
resource "aws_apigatewayv2_stage" "main" {
  api_id      = aws_apigatewayv2_api.main.id
  name        = "$default"
  auto_deploy = true
}
