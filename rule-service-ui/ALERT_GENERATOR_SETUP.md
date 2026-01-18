# Alert Generator Setup

## Prerequisites

The Alert Generator UI requires the alert-producer API server to be running.

## Starting the API Server

Before using the Alert Generator in the UI, start the API server:

```bash
cd services/alert-producer
make build-api
make run-api
```

The API server will start on port 8082.

## Verifying the API Server

Check if the API server is running:

```bash
curl http://localhost:8082/health
```

Should return: `OK`

## Troubleshooting "Failed to fetch" Errors

If you see "Failed to fetch" errors in the UI:

### 1. Check if API server is running

```bash
curl http://localhost:8082/health
```

If this fails, start the API server:
```bash
cd services/alert-producer
make run-api
```

### 2. Check Browser Console

Open browser DevTools (F12) → Console tab and look for:
- Network errors
- CORS errors
- API call logs

### 3. Check Network Tab

1. Open DevTools (F12) → Network tab
2. Try to generate alerts
3. Look for requests to `/alert-producer-api/api/v1/alerts/generate`
4. Check:
   - Status code (should be 202 Accepted)
   - Request payload
   - Response body
   - Any error messages

### 4. Common Issues

#### Issue: "Failed to fetch" or "NetworkError"
- **Cause**: API server not running or not reachable
- **Fix**: Start the API server with `make run-api` in `services/alert-producer`

#### Issue: CORS errors
- **Cause**: Browser blocking cross-origin requests
- **Fix**: The Vite proxy should handle this. Make sure you're using the dev server (`npm run dev`)

#### Issue: Connection refused
- **Cause**: API server not running on port 8082
- **Fix**: Start the API server

## Development Setup

1. **Terminal 1**: Start the API server
   ```bash
   cd services/alert-producer
   make run-api
   ```

2. **Terminal 2**: Start the UI
   ```bash
   cd rule-service-ui
   npm run dev
   ```

3. Open http://localhost:3000 and navigate to the "Alert Generator" tab

## API Endpoints

The UI uses these endpoints (proxied through Vite):
- `POST /alert-producer-api/api/v1/alerts/generate` - Start alert generation
- `GET /alert-producer-api/api/v1/alerts/generate/status?job_id=...` - Get job status
- `GET /alert-producer-api/api/v1/alerts/generate/list` - List all jobs
- `POST /alert-producer-api/api/v1/alerts/generate/stop?job_id=...` - Stop a job

The Vite proxy routes `/alert-producer-api/*` to `http://localhost:8082/*` automatically.
