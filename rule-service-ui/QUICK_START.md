# Quick Start Guide

## Starting the Rule Service

The UI requires the rule-service to be running on port 8081. To start it:

### Option 1: One-Command Setup (Recommended)

```bash
cd rule-service
make run-all
```

This will:
1. ✅ Verify Go 1.22+ is installed
2. ✅ Check Docker is installed and running
3. ✅ Install `golang-migrate` tool if missing
4. ✅ Download Go dependencies
5. ✅ Start Postgres, Kafka, and Zookeeper (if not already running)
6. ✅ Wait for services to be ready
7. ✅ Run database migrations
8. ✅ Create Kafka topics
9. ✅ Build the service
10. ✅ Start the HTTP server on port 8081

### Option 2: Manual Setup

If you prefer to set up manually:

```bash
cd rule-service

# 1. Start Docker services (Postgres, Kafka)
make up

# 2. Run database migrations
make migrate-up

# 3. Create Kafka topics
make create-topics

# 4. Run the service
make run-default
```

## Starting the UI

Once the rule-service is running, start the UI:

```bash
cd rule-service-ui
npm install  # First time only
npm run dev
```

The UI will be available at `http://localhost:3000`

## Verifying Everything Works

1. **Check rule-service is running:**
   ```bash
   curl http://localhost:8081/health
   # Should return: OK
   ```

2. **Check UI can connect:**
   - Open `http://localhost:3000`
   - You should see "✓ Connected to rule-service" at the top

3. **Test creating a client:**
   - Click "Create New Client"
   - Enter a client ID and name
   - Click "Create Client"
   - The client should appear in the table

## Troubleshooting

### Port 8081 is already in use

If you see "Port 8081 is already in use":
- Another instance of rule-service might be running
- Check: `lsof -i :8081`
- Kill the process or use a different port

### Cannot connect to rule-service

If the UI shows connection errors:
1. Verify rule-service is running: `curl http://localhost:8081/health`
2. Check rule-service logs for errors
3. Ensure Docker services (Postgres, Kafka) are running: `docker compose ps`

### Database connection errors

If you see database errors:
1. Ensure Postgres is running: `docker compose ps`
2. Check migrations ran: `cd rule-service && make db-clients`
3. Run migrations again: `make migrate-up`
