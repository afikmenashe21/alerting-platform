// API base URLs - configurable for production deployment
// In production: set VITE_API_GATEWAY_URL=https://xxx.execute-api.region.amazonaws.com
// In development: uses Vite proxy (see vite.config.js)
const API_BASE_URL = import.meta.env.VITE_API_GATEWAY_URL 
  ? `${import.meta.env.VITE_API_GATEWAY_URL}/api/v1`
  : '/api/v1';

async function handleResponse(response) {
  console.log('Response received:', {
    status: response.status,
    statusText: response.statusText,
    headers: Object.fromEntries(response.headers.entries()),
  });
  
  if (!response.ok) {
    let errorText;
    let errorMessage;
    try {
      errorText = await response.text();
      console.error('Error response body:', errorText);
      
      // Try to parse as JSON to extract error message
      try {
        const errorObj = JSON.parse(errorText);
        errorMessage = errorObj.error || errorText;
      } catch (parseErr) {
        // Not JSON, use text as-is
        errorMessage = errorText || `HTTP error! status: ${response.status}`;
      }
    } catch (e) {
      errorMessage = `HTTP error! status: ${response.status}`;
    }
    throw new Error(errorMessage || `HTTP error! status: ${response.status}`);
  }
  
  // Handle 204 No Content
  if (response.status === 204) {
    return null;
  }
  
  const contentType = response.headers.get('content-type');
  if (!contentType || !contentType.includes('application/json')) {
    const text = await response.text();
    console.warn('Non-JSON response:', text);
    return null;
  }
  
  try {
    const data = await response.json();
    console.log('Parsed JSON response:', data);
    return data;
  } catch (e) {
    console.error('Failed to parse JSON response:', e);
    throw new Error('Invalid JSON response from server');
  }
}

// ============================================================================
// Clients API
// ============================================================================

export const clientsAPI = {
  async create(clientId, name) {
    const url = `${API_BASE_URL}/clients`;
    const body = JSON.stringify({ client_id: clientId, name });
    console.log('POST', url, body);
    
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: body,
    });
    
    console.log('Response status:', response.status, response.statusText);
    return handleResponse(response);
  },

  async get(clientId) {
    const response = await fetch(`${API_BASE_URL}/clients?client_id=${clientId}`);
    return handleResponse(response);
  },

  async list() {
    const response = await fetch(`${API_BASE_URL}/clients`);
    return handleResponse(response);
  },
};

// ============================================================================
// Rules API
// ============================================================================

export const rulesAPI = {
  async create(clientId, severity, source, name) {
    const url = `${API_BASE_URL}/rules`;
    const body = JSON.stringify({ client_id: clientId, severity, source, name });
    console.log('POST', url, body);
    
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: body,
    });
    
    console.log('Response status:', response.status, response.statusText);
    return handleResponse(response);
  },

  async get(ruleId) {
    const url = `${API_BASE_URL}/rules?rule_id=${ruleId}`;
    console.log('GET', url);
    const response = await fetch(url);
    console.log('Response status:', response.status);
    return handleResponse(response);
  },

  async list(clientId = null) {
    const url = clientId 
      ? `${API_BASE_URL}/rules?client_id=${clientId}`
      : `${API_BASE_URL}/rules`;
    console.log('GET', url);
    const response = await fetch(url);
    console.log('Response status:', response.status);
    return handleResponse(response);
  },

  async update(ruleId, severity, source, name, version) {
    const response = await fetch(`${API_BASE_URL}/rules/update?rule_id=${ruleId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ severity, source, name, version }),
    });
    return handleResponse(response);
  },

  async toggle(ruleId, enabled, version) {
    const response = await fetch(`${API_BASE_URL}/rules/toggle?rule_id=${ruleId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled, version }),
    });
    return handleResponse(response);
  },

  async delete(ruleId) {
    const response = await fetch(`${API_BASE_URL}/rules/delete?rule_id=${ruleId}`, {
      method: 'DELETE',
    });
    return handleResponse(response);
  },
};

// ============================================================================
// Endpoints API
// ============================================================================

export const endpointsAPI = {
  async create(ruleId, type, value) {
    const response = await fetch(`${API_BASE_URL}/endpoints`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rule_id: ruleId, type, value }),
    });
    return handleResponse(response);
  },

  async get(endpointId) {
    const response = await fetch(`${API_BASE_URL}/endpoints?endpoint_id=${endpointId}`);
    return handleResponse(response);
  },

  async list(ruleId) {
    const response = await fetch(`${API_BASE_URL}/endpoints?rule_id=${ruleId}`);
    return handleResponse(response);
  },

  async update(endpointId, type, value) {
    const response = await fetch(`${API_BASE_URL}/endpoints/update?endpoint_id=${endpointId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ type, value }),
    });
    return handleResponse(response);
  },

  async toggle(endpointId, enabled) {
    const response = await fetch(`${API_BASE_URL}/endpoints/toggle?endpoint_id=${endpointId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    });
    return handleResponse(response);
  },

  async delete(endpointId) {
    const response = await fetch(`${API_BASE_URL}/endpoints/delete?endpoint_id=${endpointId}`, {
      method: 'DELETE',
    });
    return handleResponse(response);
  },
};

// ============================================================================
// Notifications API
// ============================================================================

export const notificationsAPI = {
  async get(notificationId) {
    const url = `${API_BASE_URL}/notifications?notification_id=${notificationId}`;
    console.log('GET', url);
    const response = await fetch(url);
    console.log('Response status:', response.status);
    return handleResponse(response);
  },

  async list(clientId = null, status = null) {
    let url = `${API_BASE_URL}/notifications`;
    const params = new URLSearchParams();
    if (clientId) params.append('client_id', clientId);
    if (status) params.append('status', status);
    if (params.toString()) {
      url += '?' + params.toString();
    }
    console.log('GET', url);
    const response = await fetch(url);
    console.log('Response status:', response.status);
    return handleResponse(response);
  },
};

// ============================================================================
// System Metrics API
// ============================================================================

export const metricsAPI = {
  async get() {
    const url = `${API_BASE_URL}/metrics`;
    console.log('GET', url);
    const response = await fetch(url);
    console.log('Response status:', response.status);
    return handleResponse(response);
  },
};

// ============================================================================
// Service Metrics API
// ============================================================================

export const serviceMetricsAPI = {
  async getAll() {
    const url = `${API_BASE_URL}/services/metrics`;
    console.log('GET', url);
    const response = await fetch(url);
    console.log('Response status:', response.status);
    return handleResponse(response);
  },

  async getService(serviceName) {
    const url = `${API_BASE_URL}/services/metrics?service=${serviceName}`;
    console.log('GET', url);
    const response = await fetch(url);
    console.log('Response status:', response.status);
    return handleResponse(response);
  },
};

// ============================================================================
// Alert Generator API (alert-producer)
// ============================================================================

// Alert producer URL - routed through API Gateway
// In production: uses same VITE_API_GATEWAY_URL with /alert-producer-api prefix
// In development: uses Vite proxy (see vite.config.js)
const ALERT_PRODUCER_API_BASE = import.meta.env.VITE_API_GATEWAY_URL 
  ? `${import.meta.env.VITE_API_GATEWAY_URL}/alert-producer-api/api/v1/alerts`
  : '/alert-producer-api/api/v1/alerts';

export const alertGeneratorAPI = {
  async generate(config) {
    const url = `${ALERT_PRODUCER_API_BASE}/generate`;
    console.log('POST', url, config);
    
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    });
    
    console.log('Response status:', response.status);
    return handleResponse(response);
  },

  async getStatus(jobId) {
    const url = `${ALERT_PRODUCER_API_BASE}/generate/status?job_id=${jobId}`;
    console.log('GET', url);
    const response = await fetch(url);
    console.log('Response status:', response.status);
    return handleResponse(response);
  },

  async list(statusFilter = null) {
    let url = `${ALERT_PRODUCER_API_BASE}/generate/list`;
    if (statusFilter) {
      url += `?status=${statusFilter}`;
    }
    console.log('GET', url);
    const response = await fetch(url);
    console.log('Response status:', response.status);
    return handleResponse(response);
  },

  async stop(jobId) {
    const url = `${ALERT_PRODUCER_API_BASE}/generate/stop?job_id=${jobId}`;
    console.log('POST', url);
    const response = await fetch(url, {
      method: 'POST',
    });
    console.log('Response status:', response.status);
    return handleResponse(response);
  },
};
