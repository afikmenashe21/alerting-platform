# Troubleshooting Guide

## Issue: Created client but nothing appears

### Step 1: Check if rule-service is running

Open a terminal and check if the service is running:

```bash
# Check if port 8081 is in use
lsof -i :8081

# Or check the health endpoint
curl http://localhost:8081/health
```

If the service is not running, start it:

```bash
cd rule-service
make run-all
```

### Step 2: Check browser console

1. Open the browser developer tools (F12)
2. Go to the Console tab
3. Look for any error messages when you:
   - Load the page
   - Create a client
   - Click Refresh

### Step 3: Check Network tab

1. Open browser developer tools (F12)
2. Go to the Network tab
3. Try creating a client
4. Look for the POST request to `/api/v1/clients`
5. Check:
   - Status code (should be 201)
   - Response body
   - Any error messages

### Step 4: Test API directly

Test the API with curl:

```bash
# List clients
curl http://localhost:8081/api/v1/clients

# Create a client
curl -X POST http://localhost:8081/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{"client_id":"test-001","name":"Test Client"}'

# List again to verify
curl http://localhost:8081/api/v1/clients
```

### Step 5: Check Vite proxy

The UI uses Vite's proxy to forward `/api` requests to `http://localhost:8081`. 

If direct API calls work but the UI doesn't:
1. Check that Vite dev server is running: `npm run dev`
2. Check the Vite console for proxy errors
3. Try accessing the API directly in the browser: `http://localhost:3000/api/v1/clients`

### Common Issues

1. **CORS errors**: Should be fixed with CORS middleware, but check browser console
2. **Port conflicts**: Make sure nothing else is using port 8081 or 3000
3. **Database not initialized**: Run migrations: `cd rule-service && make migrate-up`
4. **Service crashed**: Check rule-service logs for errors
