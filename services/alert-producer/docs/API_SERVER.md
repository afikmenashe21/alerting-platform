# Alert Producer HTTP API Server

The alert-producer HTTP API server provides a RESTful interface for generating alerts from the web UI or any HTTP client.

## Overview

The API server wraps the alert-producer CLI functionality and exposes it via HTTP endpoints. It supports:
- Starting alert generation jobs with custom configuration
- Monitoring job status in real-time
- Stopping running jobs
- Viewing job history

## Starting the API Server

### Build and Run

```bash
# Build the API server
make build-api

# Run the API server (default port 8082)
make run-api

# Run with custom port and Kafka brokers
make run-api ARGS="-port 8082 -kafka-brokers localhost:9092"
```

### Command-Line Options

- `-port`: HTTP server port (default: `8082`)
- `-kafka-brokers`: Default Kafka broker addresses (default: `localhost:9092`)

## API Endpoints

### Health Check

```
GET /health
```

Returns `OK` if the server is running.

**Response:**
```
OK
```

### Generate Alerts

```
POST /api/v1/alerts/generate
```

Starts a new alert generation job.

**Request Body:**
```json
{
  "rps": 10.0,
  "duration": "60s",
  "burst": 0,
  "seed": 0,
  "severity_dist": "HIGH:30,MEDIUM:30,LOW:25,CRITICAL:15",
  "source_dist": "api:25,db:20,cache:15,monitor:15,queue:10,worker:5,frontend:5,backend:5",
  "name_dist": "timeout:15,error:15,crash:10,slow:10,memory:10,cpu:10,disk:10,network:10,auth:5,validation:5",
  "kafka_brokers": "localhost:9092",
  "topic": "alerts.new",
  "mock": false,
  "test": false,
  "single_test": false
}
```

**All fields are optional** - defaults will be used if not specified.

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending"
}
```

**Status Code:** `202 Accepted`

### Get Job Status

```
GET /api/v1/alerts/generate/status?job_id=<job_id>
```

Retrieves the current status of a job.

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "config": {
    "rps": 10.0,
    "duration": "60s"
  },
  "created_at": "2024-01-15T10:30:00Z",
  "started_at": "2024-01-15T10:30:01Z",
  "completed_at": null,
  "alerts_sent": 150,
  "error": null
}
```

**Status Values:**
- `pending`: Job created but not started
- `running`: Job is currently generating alerts
- `completed`: Job finished successfully
- `failed`: Job failed with an error
- `cancelled`: Job was stopped by user

**Status Code:** `200 OK` or `404 Not Found`

### List Jobs

```
GET /api/v1/alerts/generate/list?status=<status>
```

Lists all jobs, optionally filtered by status.

**Query Parameters:**
- `status` (optional): Filter by status (`pending`, `running`, `completed`, `failed`, `cancelled`)

**Response:**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "config": {...},
    "created_at": "2024-01-15T10:30:00Z",
    "started_at": "2024-01-15T10:30:01Z",
    "completed_at": "2024-01-15T10:31:00Z",
    "alerts_sent": 600,
    "error": null
  }
]
```

**Status Code:** `200 OK`

### Stop Job

```
POST /api/v1/alerts/generate/stop?job_id=<job_id>
```

Stops a running job.

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "cancelled",
  ...
}
```

**Status Code:** `200 OK` or `404 Not Found`

## Configuration Options

All configuration options from the CLI are supported via the API:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `rps` | float | 10.0 | Alerts per second |
| `duration` | string | "60s" | Duration (e.g., "60s", "5m") |
| `burst` | int | 0 | Burst mode: send N alerts immediately (0 = continuous) |
| `seed` | int64 | 0 | Random seed (0 = random) |
| `severity_dist` | string | "HIGH:30,MEDIUM:30,LOW:25,CRITICAL:15" | Severity distribution |
| `source_dist` | string | "api:25,db:20,..." | Source distribution |
| `name_dist` | string | "timeout:15,error:15,..." | Name distribution |
| `kafka_brokers` | string | "localhost:9092" | Kafka broker addresses |
| `topic` | string | "alerts.new" | Kafka topic name |
| `mock` | bool | false | Use mock producer (no Kafka) |
| `test` | bool | false | Test mode (includes test alert) |
| `single_test` | bool | false | Send only one test alert |

## Example Usage

### Using curl

```bash
# Start a job
curl -X POST http://localhost:8082/api/v1/alerts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "rps": 20,
    "duration": "2m"
  }'

# Check status
curl http://localhost:8082/api/v1/alerts/generate/status?job_id=<job_id>

# Stop a job
curl -X POST http://localhost:8082/api/v1/alerts/generate/stop?job_id=<job_id>

# List all jobs
curl http://localhost:8082/api/v1/alerts/generate/list
```

### Using JavaScript

```javascript
// Start a job
const response = await fetch('http://localhost:8082/api/v1/alerts/generate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    rps: 20,
    duration: '2m'
  })
});

const { job_id } = await response.json();

// Poll for status
const status = await fetch(
  `http://localhost:8082/api/v1/alerts/generate/status?job_id=${job_id}`
);
const job = await status.json();
console.log('Job status:', job.status);
```

## CORS

The API server includes CORS headers to allow requests from web browsers. All origins are allowed by default.

## Integration with rule-service-ui

The API server is designed to be used by the rule-service-ui React application. See the UI component documentation for details on how to use it from the web interface.

## Error Handling

All errors are returned as JSON with the following format:

```json
{
  "error": "Error message here"
}
```

Common error scenarios:
- `400 Bad Request`: Invalid request body or parameters
- `404 Not Found`: Job ID not found
- `405 Method Not Allowed`: Wrong HTTP method used
