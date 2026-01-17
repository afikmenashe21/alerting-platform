# Troubleshooting Rules - Can't Add or List Rules

## Quick Checks

### 1. Is rule-service running?

```bash
curl http://localhost:8081/health
```

Should return: `OK`

If not, start it:
```bash
cd rule-service
make run-all
```

### 2. Check Browser Console

Open browser DevTools (F12) → Console tab and look for:
- Network errors
- CORS errors
- API call logs (I added console.log statements)

### 3. Test API Directly

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

### 4. Check Network Tab

1. Open DevTools (F12) → Network tab
2. Try to create a rule in the UI
3. Look for the POST request to `/api/v1/rules`
4. Check:
   - Status code (should be 201 Created)
   - Request payload
   - Response body
   - Any error messages

### 5. Common Issues

#### Issue: "Failed to load rules: ..."
- **Cause**: API not responding or wrong URL
- **Fix**: Check rule-service is running on port 8081

#### Issue: "Invalid request body"
- **Cause**: JSON parsing error or missing required fields
- **Fix**: Check browser console for the actual request body being sent

#### Issue: "Client not found"
- **Cause**: Trying to create rule with non-existent client_id
- **Fix**: Create the client first, then create the rule

#### Issue: Rules list is empty but rules exist
- **Cause**: Response might not be an array
- **Fix**: Check browser console for "Rules API response:" log

### 6. Debug Steps

The UI now has enhanced logging. Check the browser console for:

```
Loading rules, clientId: null
GET /api/v1/rules
Response status: 200
Rules API response: [...]
Loaded 3 rules
```

If you see errors, they will show:
- The exact API URL being called
- The response status
- The response body
- Any parsing errors

### 7. Verify Vite Proxy

The UI uses Vite proxy. Check `vite.config.js`:

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

### 8. Check CORS

CORS middleware should be enabled. If you see CORS errors:
1. Check `cmd/rule-service/main.go` has CORS middleware
2. Check browser console for CORS errors
3. Verify `Access-Control-Allow-Origin: *` header in response
