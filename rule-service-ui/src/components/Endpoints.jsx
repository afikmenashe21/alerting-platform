import { useState, useEffect, useCallback } from 'react';
import { endpointsAPI, rulesAPI } from '../services/api';

const ENDPOINT_TYPES = ['email', 'webhook', 'slack'];
const PAGE_SIZE_OPTIONS = [25, 50, 100, 200];

export default function Endpoints() {
  const [endpoints, setEndpoints] = useState([]);
  const [rules, setRules] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [showForm, setShowForm] = useState(false);
  const [editingEndpoint, setEditingEndpoint] = useState(null);
  const [filterRuleId, setFilterRuleId] = useState('');
  const [formData, setFormData] = useState({
    rule_id: '',
    type: 'email',
    value: '',
  });

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [totalCount, setTotalCount] = useState(0);

  const totalPages = Math.ceil(totalCount / pageSize);

  useEffect(() => {
    loadRules();
  }, []);

  const loadRules = async () => {
    try {
      // Load only 100 rules for dropdown - sufficient for most use cases
      // and reduces database load from COUNT(*) queries
      const data = await rulesAPI.list(null, 100, 0);
      setRules(data?.rules || []);
    } catch (err) {
      console.error('Failed to load rules:', err);
    }
  };

  const loadEndpoints = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const ruleId = filterRuleId || null;
      const offset = (currentPage - 1) * pageSize;
      console.log('Loading endpoints, ruleId:', ruleId, 'limit:', pageSize, 'offset:', offset);
      const data = await endpointsAPI.list(ruleId, pageSize, offset);
      console.log('Endpoints API response:', data);
      
      if (data && data.endpoints) {
        setEndpoints(data.endpoints || []);
        setTotalCount(data.total || 0);
        console.log(`Loaded ${data.endpoints?.length || 0} of ${data.total} endpoints`);
      } else if (Array.isArray(data)) {
        // Fallback for old API format
        setEndpoints(data);
        setTotalCount(data.length);
      } else {
        console.warn('Unexpected response format:', typeof data, data);
        setEndpoints([]);
        setTotalCount(0);
      }
    } catch (err) {
      console.error('Error loading endpoints:', err);
      setError(err.message);
      setEndpoints([]);
      setTotalCount(0);
    } finally {
      setLoading(false);
    }
  }, [filterRuleId, currentPage, pageSize]);

  useEffect(() => {
    loadEndpoints();
  }, [loadEndpoints]);

  // Reset to page 1 when filters or page size change
  useEffect(() => {
    setCurrentPage(1);
  }, [filterRuleId, pageSize]);

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

  const handleCreate = async (e) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    try {
      await endpointsAPI.create(formData.rule_id, formData.type, formData.value);
      setSuccess('Endpoint created successfully!');
      resetForm();
      loadEndpoints();
    } catch (err) {
      setError(err.message);
    }
  };

  const handleUpdate = async (e) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    try {
      await endpointsAPI.update(
        editingEndpoint.endpoint_id,
        formData.type,
        formData.value
      );
      setSuccess('Endpoint updated successfully!');
      resetForm();
      loadEndpoints();
    } catch (err) {
      setError(err.message);
    }
  };

  const handleToggle = async (endpoint) => {
    setError(null);
    setSuccess(null);

    try {
      await endpointsAPI.toggle(endpoint.endpoint_id, !endpoint.enabled);
      setSuccess(`Endpoint ${endpoint.enabled ? 'disabled' : 'enabled'} successfully!`);
      loadEndpoints();
    } catch (err) {
      setError(err.message);
    }
  };

  const handleDelete = async (endpointId) => {
    if (!confirm('Are you sure you want to delete this endpoint?')) {
      return;
    }

    setError(null);
    setSuccess(null);

    try {
      await endpointsAPI.delete(endpointId);
      setSuccess('Endpoint deleted successfully!');
      loadEndpoints();
    } catch (err) {
      setError(err.message);
    }
  };

  const startEdit = (endpoint) => {
    setEditingEndpoint(endpoint);
    setFormData({
      rule_id: endpoint.rule_id,
      type: endpoint.type,
      value: endpoint.value,
    });
    setShowForm(true);
  };

  const resetForm = () => {
    setFormData({ rule_id: filterRuleId || '', type: 'email', value: '' });
    setEditingEndpoint(null);
    setShowForm(false);
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <div className="card">
      <h2>Endpoints</h2>

      {error && <div className="error">Error: {error}</div>}
      {success && <div className="success">{success}</div>}

      <div style={{ marginBottom: '20px', display: 'flex', gap: '15px', flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <div className="form-group" style={{ maxWidth: '400px', marginBottom: 0 }}>
          <label>Filter by Rule</label>
          <select
            value={filterRuleId}
            onChange={(e) => {
              setFilterRuleId(e.target.value);
              setFormData({ ...formData, rule_id: e.target.value });
            }}
          >
            <option value="">All Rules</option>
            {rules.map((rule) => (
              <option key={rule.rule_id} value={rule.rule_id}>
                {rule.client_id} - {rule.severity}/{rule.source}/{rule.name}
              </option>
            ))}
          </select>
        </div>
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
        <button
          className="btn btn-primary"
          onClick={() => {
            resetForm();
            setShowForm(true);
          }}
        >
          {showForm ? 'Cancel' : 'Create New Endpoint'}
        </button>
        <button className="btn btn-secondary" onClick={loadEndpoints}>
          Refresh
        </button>
      </div>

      {/* Pagination info */}
      {totalCount > 0 && (
        <div style={{ marginBottom: '15px', color: '#666', fontSize: '14px' }}>
          Showing {((currentPage - 1) * pageSize) + 1} - {Math.min(currentPage * pageSize, totalCount)} of {totalCount} endpoints
        </div>
      )}

      {showForm && (
        <form
          onSubmit={editingEndpoint ? handleUpdate : handleCreate}
          style={{ marginTop: '20px' }}
        >
          {!editingEndpoint && (
            <div className="form-group">
              <label>Rule *</label>
              <select
                value={formData.rule_id}
                onChange={(e) => setFormData({ ...formData, rule_id: e.target.value })}
                required
              >
                <option value="">Select a rule</option>
                {rules.map((rule) => (
                  <option key={rule.rule_id} value={rule.rule_id}>
                    {rule.client_id} - {rule.severity}/{rule.source}/{rule.name}
                  </option>
                ))}
              </select>
            </div>
          )}
          <div className="form-group">
            <label>Type *</label>
            <select
              value={formData.type}
              onChange={(e) => setFormData({ ...formData, type: e.target.value })}
              required
            >
              {ENDPOINT_TYPES.map((type) => (
                <option key={type} value={type}>
                  {type.charAt(0).toUpperCase() + type.slice(1)}
                </option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label>Value *</label>
            <input
              type="text"
              value={formData.value}
              onChange={(e) => setFormData({ ...formData, value: e.target.value })}
              required
              placeholder={
                formData.type === 'email'
                  ? 'e.g., user@example.com'
                  : formData.type === 'webhook'
                  ? 'e.g., https://example.com/webhook'
                  : 'e.g., #channel or @user'
              }
            />
          </div>
          <div className="button-group">
            <button type="submit" className="btn btn-primary">
              {editingEndpoint ? 'Update Endpoint' : 'Create Endpoint'}
            </button>
            {editingEndpoint && (
              <button type="button" className="btn btn-secondary" onClick={resetForm}>
                Cancel
              </button>
            )}
          </div>
        </form>
      )}

      {loading ? (
        <div className="loading">Loading endpoints...</div>
      ) : endpoints.length === 0 ? (
        <div className="empty-state">
          <p>No endpoints found. {filterRuleId ? 'Create one for this rule.' : 'Create one to get started.'}</p>
        </div>
      ) : (
        <>
          <div style={{ overflowX: 'auto' }}>
            <table className="table">
              <thead>
                <tr>
                  <th>Endpoint ID</th>
                  <th>Rule ID</th>
                  <th>Type</th>
                  <th>Value</th>
                  <th>Enabled</th>
                  <th>Created At</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {endpoints.map((endpoint) => (
                  <tr key={endpoint.endpoint_id}>
                    <td style={{ fontSize: '12px' }}>{endpoint.endpoint_id}</td>
                    <td style={{ fontSize: '12px' }}>{endpoint.rule_id}</td>
                    <td>
                      <span className="badge badge-success">{endpoint.type}</span>
                    </td>
                    <td>{endpoint.value}</td>
                    <td>
                      <span
                        className={
                          endpoint.enabled ? 'badge badge-success' : 'badge badge-danger'
                        }
                      >
                        {endpoint.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>
                    <td>{formatDate(endpoint.created_at)}</td>
                    <td>
                      <div style={{ display: 'flex', gap: '5px' }}>
                        <button
                          className="btn btn-small btn-primary"
                          onClick={() => startEdit(endpoint)}
                        >
                          Edit
                        </button>
                        <button
                          className="btn btn-small btn-warning"
                          onClick={() => handleToggle(endpoint)}
                        >
                          {endpoint.enabled ? 'Disable' : 'Enable'}
                        </button>
                        <button
                          className="btn btn-small btn-danger"
                          onClick={() => handleDelete(endpoint.endpoint_id)}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

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
