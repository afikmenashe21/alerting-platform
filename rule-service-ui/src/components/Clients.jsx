import { useState, useEffect } from 'react';
import { clientsAPI } from '../services/api';

export default function Clients() {
  const [clients, setClients] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({ client_id: '', name: '' });
  const [connectionStatus, setConnectionStatus] = useState('checking');

  useEffect(() => {
    // Test connection first
    testConnection();
    loadClients();
  }, []);

  const testConnection = async () => {
    try {
      // Try API endpoint through proxy
      const response = await fetch('/api/v1/clients');
      console.log('Connection test - Status:', response.status);
      
      // Any response (even 200 with empty array) means service is reachable
      if (response.status === 0) {
        // Network error - service not running
        setConnectionStatus('error');
      } else {
        // Got a response (even if it's an error) - service is running
        setConnectionStatus('connected');
      }
    } catch (err) {
      console.error('Connection test failed:', err);
      setConnectionStatus('error');
      // Don't set error here, just connection status - let individual operations show their errors
    }
  };

  const loadClients = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await clientsAPI.list();
      console.log('Clients API response:', data);
      if (Array.isArray(data)) {
        setClients(data);
      } else {
        console.warn('Expected array but got:', typeof data, data);
        setClients([]);
      }
    } catch (err) {
      console.error('Error loading clients:', err);
      setError(err.message);
      setClients([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setLoading(true);

    try {
      console.log('Creating client:', formData);
      const clientId = formData.client_id;
      const clientName = formData.name;
      
      const result = await clientsAPI.create(clientId, clientName);
      console.log('Client created, response:', result);
      
      // Show success message before clearing form
      setSuccess(`Client "${clientName}" (${clientId}) created successfully!`);
      
      // Clear form and hide it
      setFormData({ client_id: '', name: '' });
      setShowForm(false);
      
      // Reload clients list immediately
      await loadClients();
    } catch (err) {
      console.error('Error creating client:', err);
      const errorMsg = err.message || 'Unknown error occurred';
      setError(`Failed to create client: ${errorMsg}`);
      // Keep form open so user can retry
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <div className="card">
      <h2>Clients</h2>

      {connectionStatus === 'checking' && (
        <div className="loading" style={{ marginBottom: '10px' }}>Checking connection to rule-service...</div>
      )}
      {connectionStatus === 'error' && (
        <div className="error" style={{ marginBottom: '10px' }}>
          ⚠️ Connection Error: Cannot reach rule-service at http://localhost:8081
          <br />
          <small style={{ fontSize: '12px' }}>
            Please ensure the rule-service is running: <code>cd rule-service && make run-all</code>
            <br />
            Or test directly: <code>curl http://localhost:8081/health</code>
          </small>
        </div>
      )}
      {connectionStatus === 'connected' && (
        <div className="success" style={{ marginBottom: '10px', padding: '5px 15px', fontSize: '12px' }}>
          ✓ Connected to rule-service
        </div>
      )}

      {error && <div className="error">Error: {error}</div>}
      {success && <div className="success">{success}</div>}

      <div className="button-group">
        <button className="btn btn-primary" onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Cancel' : 'Create New Client'}
        </button>
        <button className="btn btn-secondary" onClick={loadClients}>
          Refresh
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} style={{ marginTop: '20px' }}>
          <div className="form-group">
            <label>Client ID *</label>
            <input
              type="text"
              value={formData.client_id}
              onChange={(e) => setFormData({ ...formData, client_id: e.target.value })}
              required
              placeholder="e.g., client-001"
            />
          </div>
          <div className="form-group">
            <label>Name *</label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
              placeholder="e.g., Acme Corp"
            />
          </div>
          <div className="button-group">
            <button type="submit" className="btn btn-primary">
              Create Client
            </button>
          </div>
        </form>
      )}

      {loading ? (
        <div className="loading">Loading clients...</div>
      ) : clients.length === 0 ? (
        <div className="empty-state">
          <p>No clients found. Create one to get started.</p>
        </div>
      ) : (
        <table className="table">
          <thead>
            <tr>
              <th>Client ID</th>
              <th>Name</th>
              <th>Created At</th>
              <th>Updated At</th>
            </tr>
          </thead>
          <tbody>
            {clients.map((client) => (
              <tr key={client.client_id}>
                <td>{client.client_id}</td>
                <td>{client.name}</td>
                <td>{formatDate(client.created_at)}</td>
                <td>{formatDate(client.updated_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
