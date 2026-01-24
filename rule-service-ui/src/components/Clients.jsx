import { useState, useEffect, useCallback } from 'react';
import { clientsAPI } from '../services/api';

const PAGE_SIZE_OPTIONS = [25, 50, 100, 200];

export default function Clients() {
  const [clients, setClients] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({ client_id: '', name: '' });
  const [connectionStatus, setConnectionStatus] = useState('checking');
  
  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [totalCount, setTotalCount] = useState(0);

  const totalPages = Math.ceil(totalCount / pageSize);

  useEffect(() => {
    // Test connection first
    testConnection();
  }, []);

  const testConnection = async () => {
    try {
      // Try API endpoint through proxy
      const response = await fetch('/api/v1/clients?limit=1');
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

  const loadClients = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const offset = (currentPage - 1) * pageSize;
      console.log('Loading clients, limit:', pageSize, 'offset:', offset);
      const data = await clientsAPI.list(pageSize, offset);
      console.log('Clients API response:', data);
      
      if (data && data.clients) {
        setClients(data.clients || []);
        setTotalCount(data.total || 0);
        console.log(`Loaded ${data.clients?.length || 0} of ${data.total} clients`);
      } else if (Array.isArray(data)) {
        // Fallback for old API format
        setClients(data);
        setTotalCount(data.length);
      } else {
        console.warn('Unexpected response format:', typeof data, data);
        setClients([]);
        setTotalCount(0);
      }
    } catch (err) {
      console.error('Error loading clients:', err);
      setError(err.message);
      setClients([]);
      setTotalCount(0);
    } finally {
      setLoading(false);
    }
  }, [currentPage, pageSize]);

  useEffect(() => {
    loadClients();
  }, [loadClients]);

  // Reset to page 1 when page size changes
  useEffect(() => {
    setCurrentPage(1);
  }, [pageSize]);

  const handlePageChange = (newPage) => {
    if (newPage >= 1 && newPage <= totalPages) {
      setCurrentPage(newPage);
    }
  };

  const handlePageSizeChange = (newSize) => {
    setPageSize(newSize);
    setCurrentPage(1);
  };

  // Generate page numbers to display
  const getPageNumbers = () => {
    const pages = [];
    const maxVisible = 5;
    
    if (totalPages <= maxVisible) {
      for (let i = 1; i <= totalPages; i++) pages.push(i);
    } else {
      if (currentPage <= 3) {
        for (let i = 1; i <= 4; i++) pages.push(i);
        pages.push('...');
        pages.push(totalPages);
      } else if (currentPage >= totalPages - 2) {
        pages.push(1);
        pages.push('...');
        for (let i = totalPages - 3; i <= totalPages; i++) pages.push(i);
      } else {
        pages.push(1);
        pages.push('...');
        for (let i = currentPage - 1; i <= currentPage + 1; i++) pages.push(i);
        pages.push('...');
        pages.push(totalPages);
      }
    }
    return pages;
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

      <div style={{ marginBottom: '20px', display: 'flex', gap: '15px', flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <div className="form-group" style={{ maxWidth: '120px', marginBottom: 0 }}>
          <label>Per Page</label>
          <select
            value={pageSize}
            onChange={(e) => handlePageSizeChange(Number(e.target.value))}
          >
            {PAGE_SIZE_OPTIONS.map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </div>
        <button className="btn btn-primary" onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Cancel' : 'Create New Client'}
        </button>
        <button className="btn btn-secondary" onClick={loadClients}>
          Refresh
        </button>
      </div>

      {/* Pagination info */}
      {totalCount > 0 && (
        <div style={{ marginBottom: '15px', color: '#666', fontSize: '14px' }}>
          Showing {((currentPage - 1) * pageSize) + 1} - {Math.min(currentPage * pageSize, totalCount)} of {totalCount} clients
        </div>
      )}

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
        <>
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

          {/* Pagination controls */}
          {totalPages > 1 && (
            <div style={{ 
              display: 'flex', 
              justifyContent: 'center', 
              alignItems: 'center', 
              gap: '8px', 
              marginTop: '20px',
              flexWrap: 'wrap'
            }}>
              <button
                className="btn btn-secondary"
                onClick={() => handlePageChange(1)}
                disabled={currentPage === 1}
                style={{ padding: '6px 12px' }}
              >
                First
              </button>
              <button
                className="btn btn-secondary"
                onClick={() => handlePageChange(currentPage - 1)}
                disabled={currentPage === 1}
                style={{ padding: '6px 12px' }}
              >
                Prev
              </button>
              
              {getPageNumbers().map((page, idx) => (
                page === '...' ? (
                  <span key={`ellipsis-${idx}`} style={{ padding: '6px 8px', color: '#666' }}>...</span>
                ) : (
                  <button
                    key={page}
                    className={`btn ${page === currentPage ? 'btn-primary' : 'btn-secondary'}`}
                    onClick={() => handlePageChange(page)}
                    style={{ 
                      padding: '6px 12px',
                      minWidth: '40px',
                      fontWeight: page === currentPage ? 'bold' : 'normal'
                    }}
                  >
                    {page}
                  </button>
                )
              ))}
              
              <button
                className="btn btn-secondary"
                onClick={() => handlePageChange(currentPage + 1)}
                disabled={currentPage === totalPages}
                style={{ padding: '6px 12px' }}
              >
                Next
              </button>
              <button
                className="btn btn-secondary"
                onClick={() => handlePageChange(totalPages)}
                disabled={currentPage === totalPages}
                style={{ padding: '6px 12px' }}
              >
                Last
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
