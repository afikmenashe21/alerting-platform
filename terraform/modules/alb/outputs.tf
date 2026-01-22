output "alb_arn" {
  description = "ARN of the Application Load Balancer"
  value       = aws_lb.main.arn
}

output "alb_dns_name" {
  description = "DNS name of the Application Load Balancer"
  value       = aws_lb.main.dns_name
}

output "alb_zone_id" {
  description = "Zone ID of the Application Load Balancer"
  value       = aws_lb.main.zone_id
}

output "rule_service_target_group_arn" {
  description = "ARN of the rule-service target group"
  value       = aws_lb_target_group.rule_service.arn
}

output "alert_producer_target_group_arn" {
  description = "ARN of the alert-producer target group"
  value       = aws_lb_target_group.alert_producer.arn
}

output "alb_security_group_id" {
  description = "Security group ID for ALB"
  value       = aws_security_group.alb.id
}
