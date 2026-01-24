# Troubleshooting Guide

## General Issues

### Issue: Created client but nothing appears

#### Step 1: Check if rule-service is running

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

#### Step 2: Check browser console

1. Open the browser developer tools (F12)
2. Go to the Console tab
3. Look for any error messages when you:
   - Load the page
   - Create a client
   - Click Refresh

#### Step 3: Check Network tab

1. Open browser developer tools (F12)
2. Go to the Network tab
3. Try creating a client
4. Look for the POST request to `/api/v1/clients`
5. Check:
   - Status code (should be 201)
   - Response body
   - Any error messages

#### Step 4: Test API directly

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

#### Step 5: Check Vite proxy

The UI uses Vite's proxy to forward `/api` requests to `http://localhost:8081`.

If direct API calls work but the UI doesn't:
1. Check that Vite dev server is running: `npm run dev`
2. Check the Vite console for proxy errors
3. Try accessing the API directly in the browser: `http://localhost:3000/api/v1/clients`

#### Common Issues

1. **CORS errors**: Should be fixed with CORS middleware, but check browser console
2. **Port conflicts**: Make sure nothing else is using port 8081 or 3000
3. **Database not initialized**: Run migrations: `cd rule-service && make migrate-up`
4. **Service crashed**: Check rule-service logs for errors

---

## Rules Issues

### Issue: Can't add or list rules

#### Quick Checks

1. **Is rule-service running?**
   ```bash
   curl http://localhost:8081/health
   ```
   Should return: `OK`

2. **Test API directly**:
   ```bash
   # List rules
   curl http://localhost:8081/api/v1/rules

   # Create a client first
   curl -X POST http://localhost:8081/api/v1/clients \
     -H "Content-Type: application/json" \
     -d '{"client_id": "test", "name": "Test"}'

   # Create a rule
   curl -X POST http://localhost:8081/api/v1/rules \
     -H "Content-Type: application/json" \
     -d '{"client_id": "test", "severity": "HIGH", "source": "api", "name": "test"}'

   # List rules again
   curl http://localhost:8081/api/v1/rules
   ```

#### Common Rule Errors

- **"Failed to load rules: ..."** — API not responding or wrong URL. Check rule-service is running on port 8081.
- **"Invalid request body"** — JSON parsing error or missing required fields. Check browser console for the actual request body being sent.
- **"Client not found"** — Trying to create rule with non-existent client_id. Create the client first.
- **Rules list is empty but rules exist** — Response might not be an array. Check browser console for "Rules API response:" log.

#### Debug Logging

The UI has enhanced logging. Check the browser console for:
```
Loading rules, clientId: null
GET /api/v1/rules
Response status: 200
Rules API response: [...]
Loaded 3 rules
```

---

## Notifications Issues

### Issue: 404 error on notifications endpoint

The rule-service needs to be restarted with the latest code.

1. **Stop the current rule-service** (if running):
   - Press `Ctrl+C` in the terminal where it's running
   - Or find the process: `lsof -i :8081` and kill it

2. **Rebuild and restart**:
   ```bash
   cd rule-service
   make run-all
   ```

3. **Verify the endpoint is available**:
   ```bash
   curl http://localhost:8081/api/v1/notifications
   ```
   You should get either an empty array `[]` or a JSON array of notifications, NOT a 404.

4. **Test with filters**:
   ```bash
   curl "http://localhost:8081/api/v1/notifications?status=RECEIVED"
   curl "http://localhost:8081/api/v1/notifications?client_id=client-1"
   ```

If curl works but the UI doesn't, it's a frontend issue. If curl also returns 404, the service needs to be rebuilt.

---

## Vite Proxy Configuration

The UI uses Vite proxy to forward API requests. Check `vite.config.js`:

```js
proxy: {
  '/api': {
    target: 'http://localhost:8081',
    changeOrigin: true,
  },
}
```

If the proxy isn't working:
1. Restart Vite dev server: `npm run dev`
2. Check Vite console for proxy errors
3. Try accessing API directly: `http://localhost:3000/api/v1/rules`

## CORS

CORS middleware should be enabled. If you see CORS errors:
1. Check `cmd/rule-service/main.go` has CORS middleware
2. Check browser console for CORS errors
3. Verify `Access-Control-Allow-Origin: *` header in response
