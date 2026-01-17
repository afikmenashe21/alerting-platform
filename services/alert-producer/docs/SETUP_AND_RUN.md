# Getting Started

This guide covers all ways to get alert-producer up and running.

## 🚀 Quickest Way (Recommended)

**One command does everything:**

```bash
make setup-run
```

This automatically:
1. ✅ Checks prerequisites (Go 1.22+, Docker)
2. ✅ Downloads dependencies
3. ✅ Starts Kafka (if needed)
4. ✅ Creates topics
5. ✅ Builds the service
6. ✅ Runs it

## Prerequisites

- **Go 1.22+** - Check with `go version`
- **Docker Desktop** - Must be installed and running

## Usage Examples

### Basic Usage (Default: 10 RPS for 60 seconds)
```bash
make setup-run
```

### Custom Rate and Duration
```bash
make setup-run ARGS="-rps 50 -duration 5m"
```

### Burst Mode
```bash
make setup-run ARGS="-burst 1000"
```

### Mock Mode (No Kafka Required)
```bash
make setup-run ARGS="--mock -burst 10"
```

### All Custom Options
```bash
make setup-run ARGS="-rps 20 -duration 2m -severity-dist 'HIGH:50,MEDIUM:30,LOW:20' -kafka-brokers localhost:9092"
```

## Alternative: Manual Setup

If you prefer manual control or are using centralized infrastructure:

### Step 1: Start Infrastructure

```bash
# Using centralized infrastructure (recommended)
cd ../../ && make setup-infra

# Or manually start Kafka
docker compose up -d
```

Wait 10-15 seconds for Kafka to be ready.

### Step 2: Verify Kafka

```bash
# Check status
docker compose ps

# Or use Makefile
make kafka-status
```

### Step 3: Create Topic (if needed)

Topics are usually auto-created, but you can create manually:

```bash
docker exec kafka kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic alerts.new \
  --partitions 3 \
  --replication-factor 1 \
  --if-not-exists
```

### Step 4: Build and Run

```bash
# Build
make build

# Run with defaults (10 RPS for 60 seconds)
make run

# Or with custom parameters
./bin/alert-producer -rps 50 -duration 5m -kafka-brokers localhost:9092

# Burst mode (send N alerts immediately)
./bin/alert-producer -burst 1000 -kafka-brokers localhost:9092
```

### Step 5: Verify Messages

**Option 1: Kafka UI**
Open http://localhost:8080 and navigate to `alerts.new` topic.

**Option 2: Command Line**
```bash
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic alerts.new \
  --from-beginning \
  --max-messages 5
```

## Direct Script Usage

You can also run scripts directly:

```bash
# Run service (infrastructure should be started separately)
./scripts/run-all.sh

# With custom arguments
./scripts/run-all.sh -rps 50 -duration 5m

# Start infrastructure first (from root directory)
cd ../.. && make setup-infra
```

## Environment Variables

You can also set defaults via environment variables:

```bash
export KAFKA_BROKERS=localhost:9092
export RPS=20
export DURATION=5m
export BURST=0
export MOCK_MODE=false

make setup-run
```

## What It Checks

### Prerequisites
- ✅ **Go 1.22+** - Checks version and fails if not installed or too old
- ✅ **Docker** - Verifies Docker is installed and running
- ✅ **Dependencies** - Downloads and tidies Go modules

### Infrastructure
- ✅ **Kafka Containers** - Starts if not running
- ✅ **Kafka Readiness** - Waits up to 60 seconds for Kafka to be ready
- ✅ **Topic Creation** - Creates `alerts.new` topic if it doesn't exist

### Build & Run
- ✅ **Compilation** - Builds the binary
- ✅ **Execution** - Runs with your specified parameters

## Troubleshooting

### Go Not Found
```bash
# Install Go from https://go.dev/dl/
# Or use a version manager like gvm or asdf
```

### Docker Not Running
```bash
# Start Docker Desktop
# On macOS: Open Docker Desktop app
# On Linux: sudo systemctl start docker
```

### Kafka Won't Start
```bash
# Check Docker status
docker ps

# Check logs
make kafka-logs

# Try manual start
docker compose up -d
```

### Port Already in Use
```bash
# Check what's using port 9092
lsof -i :9092

# Stop conflicting service or change port in root-level docker-compose.yml
```

## Test Without Kafka (Mock Mode)

For testing without Kafka infrastructure:

```bash
./bin/alert-producer --mock -burst 10
```

This logs alerts to console instead of publishing to Kafka.

## Stop Kafka

When done testing:

```bash
make kafka-down
# or
docker compose down
```

## Exit Codes

The script exits with:
- `0` - Success (service is running)
- `1` - Failure (check error messages)

## Integration with CI/CD

The script is designed to be CI/CD friendly:

```yaml
# Example GitHub Actions
- name: Setup and Run Tests
  run: |
    make setup-run ARGS="-burst 100"
```

The script will fail fast if prerequisites are missing, making it easy to debug in CI environments.
