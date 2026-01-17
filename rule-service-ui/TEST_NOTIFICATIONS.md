# Testing Notifications Endpoint

If you're getting a 404 error, the rule-service needs to be restarted with the new code.

## Steps to Fix

1. **Stop the current rule-service** (if running):
   - Press `Ctrl+C` in the terminal where it's running
   - Or find the process: `lsof -i :8081` and kill it

2. **Rebuild and restart the rule-service**:
   ```bash
   cd rule-service
   make run-all
   ```

3. **Verify the endpoint is available**:
   ```bash
   curl http://localhost:8081/api/v1/notifications
   ```
   
   You should get either:
   - An empty array `[]` if no notifications exist
   - A JSON array of notifications
   - NOT a 404 error

4. **If you still get 404**, check:
   - Is the service actually running? `curl http://localhost:8081/health`
   - Are there any build errors? Check the rule-service logs
   - Did the code compile? The new notification handlers need to be in the binary

## Quick Test

Test the endpoint directly:
```bash
# Should return [] or a list of notifications
curl http://localhost:8081/api/v1/notifications

# Test with filters
curl "http://localhost:8081/api/v1/notifications?status=RECEIVED"
curl "http://localhost:8081/api/v1/notifications?client_id=client-1"
```

If curl works but the UI doesn't, it's a frontend issue. If curl also returns 404, the service needs to be restarted.
