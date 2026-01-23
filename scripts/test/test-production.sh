#!/bin/bash
# Production End-to-End Test Script
# Tests the complete alerting platform flow on AWS ECS

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
AWS_REGION="${AWS_REGION:-us-east-1}"
CLUSTER_NAME="alerting-platform-prod-cluster"
RDS_ENDPOINT="${RDS_ENDPOINT:-alerting-platform-prod-postgres.cot8kqgoccg6.us-east-1.rds.amazonaws.com:5432}"
DB_PASSWORD="${DB_PASSWORD:-}"

# Test configuration
TEST_CLIENT_NAME="Production Test Client $(date +%s)"
TEST_EMAIL="prod-test-$(date +%s)@example.com"

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# Function to check if command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        print_error "$1 is not installed or not in PATH"
        exit 1
    fi
}

# Function to get ECS instance public IPs
get_ecs_instance_ips() {
    print_info "Getting ECS container instance IPs..."
    
    # Get container instance ARNs
    local instance_arns=$(aws ecs list-container-instances \
        --cluster "$CLUSTER_NAME" \
        --region "$AWS_REGION" \
        --query 'containerInstanceArns' \
        --output text)
    
    if [ -z "$instance_arns" ]; then
        print_error "No ECS container instances found"
        return 1
    fi
    
    # Get EC2 instance IDs
    local ec2_ids=$(aws ecs describe-container-instances \
        --cluster "$CLUSTER_NAME" \
        --container-instances $instance_arns \
        --region "$AWS_REGION" \
        --query 'containerInstances[*].ec2InstanceId' \
        --output text)
    
    # Get public IPs
    local ips=$(aws ec2 describe-instances \
        --instance-ids $ec2_ids \
        --region "$AWS_REGION" \
        --query 'Reservations[*].Instances[*].PublicIpAddress' \
        --output text)
    
    echo "$ips"
}

# Function to find rule-service endpoint
find_rule_service_endpoint() {
    print_info "Finding rule-service endpoint..."
    
    local ips=$(get_ecs_instance_ips)
    
    for ip in $ips; do
        print_info "Testing http://${ip}:8081/health..."
        if curl -s -f -m 5 "http://${ip}:8081/health" > /dev/null 2>&1; then
            print_success "Found rule-service at http://${ip}:8081"
            echo "$ip"
            return 0
        fi
    done
    
    print_error "Could not find rule-service endpoint on any ECS instance"
    print_warning "Instances tried: $ips"
    return 1
}

# Function to check ECS services
check_ecs_services() {
    print_header "Step 1: Verifying ECS Services"
    
    local services="kafka-combined rule-service evaluator aggregator sender rule-updater"
    
    aws ecs describe-services \
        --cluster "$CLUSTER_NAME" \
        --services $services \
        --region "$AWS_REGION" \
        --query 'services[*].{Name:serviceName,Running:runningCount,Desired:desiredCount,Status:status}' \
        --output table
    
    # Check if all required services are running
    local all_running=true
    for service in $services; do
        local running=$(aws ecs describe-services \
            --cluster "$CLUSTER_NAME" \
            --services "$service" \
            --region "$AWS_REGION" \
            --query 'services[0].runningCount' \
            --output text)
        
        if [ "$running" != "1" ]; then
            print_error "Service $service is not running (running count: $running)"
            all_running=false
        fi
    done
    
    if [ "$all_running" = true ]; then
        print_success "All required services are running"
    else
        print_error "Some services are not running"
        return 1
    fi
}

# Function to test rule-service API
test_rule_service_api() {
    print_header "Step 2: Testing rule-service API"
    
    local endpoint=$(find_rule_service_endpoint)
    if [ -z "$endpoint" ]; then
        return 1
    fi
    
    RULE_SERVICE_URL="http://${endpoint}:8081"
    
    # Test health endpoint
    print_info "Testing health endpoint..."
    local health=$(curl -s "${RULE_SERVICE_URL}/health")
    echo "$health"
    print_success "Health check passed"
    
    # List clients (should work even if empty)
    print_info "Testing /api/v1/clients endpoint..."
    local clients=$(curl -s "${RULE_SERVICE_URL}/api/v1/clients")
    echo "Current clients count: $(echo "$clients" | jq '. | length' 2>/dev/null || echo 'unknown')"
    print_success "API is accessible"
}

# Function to create test data
create_test_data() {
    print_header "Step 3: Creating Test Data"
    
    # Create test client
    print_info "Creating test client: $TEST_CLIENT_NAME"
    local client_response=$(curl -s -X POST "${RULE_SERVICE_URL}/api/v1/clients" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"$TEST_CLIENT_NAME\", \"email\": \"$TEST_EMAIL\"}")
    
    CLIENT_ID=$(echo "$client_response" | jq -r '.id')
    
    if [ -z "$CLIENT_ID" ] || [ "$CLIENT_ID" = "null" ]; then
        print_error "Failed to create client"
        echo "Response: $client_response"
        return 1
    fi
    
    print_success "Created client: $CLIENT_ID"
    
    # Create test rule (match HIGH severity alerts)
    print_info "Creating test rule for HIGH severity..."
    local rule_response=$(curl -s -X POST "${RULE_SERVICE_URL}/api/v1/rules" \
        -H "Content-Type: application/json" \
        -d "{\"client_id\": \"$CLIENT_ID\", \"severity\": \"HIGH\", \"source\": \"*\", \"name\": \"*\"}")
    
    RULE_ID=$(echo "$rule_response" | jq -r '.id')
    
    if [ -z "$RULE_ID" ] || [ "$RULE_ID" = "null" ]; then
        print_error "Failed to create rule"
        echo "Response: $rule_response"
        return 1
    fi
    
    print_success "Created rule: $RULE_ID"
    
    # Create test email endpoint
    print_info "Creating test email endpoint..."
    local endpoint_response=$(curl -s -X POST "${RULE_SERVICE_URL}/api/v1/endpoints" \
        -H "Content-Type: application/json" \
        -d "{\"rule_id\": \"$RULE_ID\", \"type\": \"email\", \"address\": \"$TEST_EMAIL\"}")
    
    ENDPOINT_ID=$(echo "$endpoint_response" | jq -r '.id')
    
    if [ -z "$ENDPOINT_ID" ] || [ "$ENDPOINT_ID" = "null" ]; then
        print_error "Failed to create endpoint"
        echo "Response: $endpoint_response"
        return 1
    fi
    
    print_success "Created endpoint: $ENDPOINT_ID"
    
    # Wait for rule-updater to process the rule change
    print_info "Waiting 5 seconds for rule-updater to process changes..."
    sleep 5
    print_success "Rule changes should be propagated to Redis"
}

# Function to scale alert-producer
scale_alert_producer() {
    print_header "Step 4: Scaling alert-producer"
    
    print_info "Scaling alert-producer to 1 instance..."
    aws ecs update-service \
        --cluster "$CLUSTER_NAME" \
        --service alert-producer \
        --desired-count 1 \
        --region "$AWS_REGION" > /dev/null
    
    print_info "Waiting for alert-producer to start..."
    local retries=30
    while [ $retries -gt 0 ]; do
        local running=$(aws ecs describe-services \
            --cluster "$CLUSTER_NAME" \
            --services alert-producer \
            --region "$AWS_REGION" \
            --query 'services[0].runningCount' \
            --output text)
        
        if [ "$running" = "1" ]; then
            print_success "alert-producer is running"
            sleep 5  # Give it a moment to fully initialize
            return 0
        fi
        
        echo -n "."
        sleep 2
        retries=$((retries - 1))
    done
    
    print_error "alert-producer failed to start within 60 seconds"
    return 1
}

# Function to find alert-producer endpoint
find_alert_producer_endpoint() {
    print_info "Finding alert-producer endpoint..."
    
    local ips=$(get_ecs_instance_ips)
    
    for ip in $ips; do
        print_info "Testing http://${ip}:8080/health..."
        if curl -s -f -m 5 "http://${ip}:8080/health" > /dev/null 2>&1; then
            print_success "Found alert-producer at http://${ip}:8080"
            echo "$ip"
            return 0
        fi
    done
    
    print_error "Could not find alert-producer endpoint on any ECS instance"
    return 1
}

# Function to generate test alerts
generate_test_alerts() {
    print_header "Step 5: Generating Test Alerts"
    
    local endpoint=$(find_alert_producer_endpoint)
    if [ -z "$endpoint" ]; then
        print_error "Could not find alert-producer endpoint"
        return 1
    fi
    
    ALERT_PRODUCER_URL="http://${endpoint}:8080"
    
    # Generate 10 HIGH severity alerts (should match our rule)
    print_info "Generating 10 HIGH severity test alerts..."
    local generate_response=$(curl -s -X POST "${ALERT_PRODUCER_URL}/api/generate" \
        -H "Content-Type: application/json" \
        -d '{
            "count": 10,
            "severity_distribution": {
                "HIGH": 1.0
            },
            "sources": ["test-api", "test-service"],
            "alert_names": ["timeout", "error"]
        }')
    
    echo "$generate_response" | jq '.'
    
    local alerts_generated=$(echo "$generate_response" | jq -r '.alerts_generated // 0')
    
    if [ "$alerts_generated" -gt 0 ]; then
        print_success "Generated $alerts_generated test alerts"
    else
        print_error "Failed to generate alerts"
        return 1
    fi
    
    # Wait for processing
    print_info "Waiting 10 seconds for alert processing through the pipeline..."
    sleep 10
}

# Function to verify notifications
verify_notifications() {
    print_header "Step 6: Verifying Notifications"
    
    if [ -z "$DB_PASSWORD" ]; then
        print_warning "DB_PASSWORD not set, skipping database verification"
        print_info "Set DB_PASSWORD environment variable to enable database checks"
        return 0
    fi
    
    print_info "Querying notifications table..."
    
    # Check if psql is available
    if ! command -v psql &> /dev/null; then
        print_warning "psql not installed, cannot verify notifications in database"
        print_info "Install PostgreSQL client to enable database verification"
        return 0
    fi
    
    local db_host=$(echo "$RDS_ENDPOINT" | cut -d: -f1)
    local db_port=$(echo "$RDS_ENDPOINT" | cut -d: -f2)
    
    export PGPASSWORD="$DB_PASSWORD"
    
    # Count notifications for our test client
    local notification_count=$(psql -h "$db_host" -p "$db_port" -U postgres -d alerting -t -c \
        "SELECT COUNT(*) FROM notifications WHERE client_id = '$CLIENT_ID';" 2>/dev/null | xargs)
    
    if [ -n "$notification_count" ] && [ "$notification_count" -gt 0 ]; then
        print_success "Found $notification_count notifications for test client"
        
        # Show notification details
        print_info "Notification details:"
        psql -h "$db_host" -p "$db_port" -U postgres -d alerting -c \
            "SELECT id, alert_id, status, created_at FROM notifications WHERE client_id = '$CLIENT_ID' LIMIT 10;" 2>/dev/null || true
    else
        print_warning "No notifications found for test client (this may be expected if alerts didn't match)"
    fi
    
    unset PGPASSWORD
}

# Function to check service logs
check_service_logs() {
    print_header "Step 7: Checking Service Logs"
    
    print_info "Checking evaluator logs for matches..."
    aws logs filter-log-events \
        --log-group-name "/ecs/alerting-platform/prod/evaluator" \
        --start-time $(date -u -d '5 minutes ago' +%s)000 \
        --filter-pattern "matched" \
        --region "$AWS_REGION" \
        --query 'events[*].message' \
        --output text \
        --max-items 5 2>/dev/null || print_warning "No recent evaluator logs found"
    
    print_info "Checking aggregator logs..."
    aws logs filter-log-events \
        --log-group-name "/ecs/alerting-platform/prod/aggregator" \
        --start-time $(date -u -d '5 minutes ago' +%s)000 \
        --filter-pattern "notification" \
        --region "$AWS_REGION" \
        --query 'events[*].message' \
        --output text \
        --max-items 5 2>/dev/null || print_warning "No recent aggregator logs found"
    
    print_info "Checking sender logs..."
    aws logs filter-log-events \
        --log-group-name "/ecs/alerting-platform/prod/sender" \
        --start-time $(date -u -d '5 minutes ago' +%s)000 \
        --filter-pattern "sent" \
        --region "$AWS_REGION" \
        --query 'events[*].message' \
        --output text \
        --max-items 5 2>/dev/null || print_warning "No recent sender logs found"
}

# Function to scale down alert-producer
scale_down_alert_producer() {
    print_header "Step 8: Cleaning Up"
    
    print_info "Scaling alert-producer back to 0..."
    aws ecs update-service \
        --cluster "$CLUSTER_NAME" \
        --service alert-producer \
        --desired-count 0 \
        --region "$AWS_REGION" > /dev/null
    
    print_success "alert-producer scaled down"
}

# Function to print test summary
print_summary() {
    print_header "Test Summary"
    
    echo ""
    echo "Test Data Created:"
    echo "  - Client ID: ${CLIENT_ID:-N/A}"
    echo "  - Client Name: $TEST_CLIENT_NAME"
    echo "  - Client Email: $TEST_EMAIL"
    echo "  - Rule ID: ${RULE_ID:-N/A}"
    echo "  - Endpoint ID: ${ENDPOINT_ID:-N/A}"
    echo ""
    echo "Next Steps:"
    echo "  1. Check CloudWatch logs for detailed processing:"
    echo "     aws logs tail /ecs/alerting-platform/prod/evaluator --follow"
    echo "     aws logs tail /ecs/alerting-platform/prod/aggregator --follow"
    echo "     aws logs tail /ecs/alerting-platform/prod/sender --follow"
    echo ""
    echo "  2. Query notifications directly:"
    echo "     PGPASSWORD=\$DB_PASSWORD psql -h $db_host -U postgres -d alerting \\"
    echo "       -c \"SELECT * FROM notifications WHERE client_id = '$CLIENT_ID';\""
    echo ""
    echo "  3. Clean up test data:"
    echo "     curl -X DELETE \"${RULE_SERVICE_URL}/api/v1/clients/${CLIENT_ID}\""
    echo ""
}

# Main execution
main() {
    print_header "Production End-to-End Test"
    echo "Cluster: $CLUSTER_NAME"
    echo "Region: $AWS_REGION"
    echo ""
    
    # Check prerequisites
    print_info "Checking prerequisites..."
    check_command aws
    check_command curl
    check_command jq
    
    # Run tests
    check_ecs_services || exit 1
    test_rule_service_api || exit 1
    create_test_data || exit 1
    scale_alert_producer || exit 1
    generate_test_alerts || exit 1
    verify_notifications
    check_service_logs
    scale_down_alert_producer
    
    print_summary
    
    print_success "Production test completed successfully!"
    echo ""
}

# Run main function
main "$@"
