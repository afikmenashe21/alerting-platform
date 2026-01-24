import { useState, useEffect, useCallback } from 'react';
import { rulesAPI, clientsAPI } from '../services/api';

const SEVERITY_OPTIONS = ['LOW', 'MEDIUM', 'HIGH', 'CRITICAL'];
const PAGE_SIZE_OPTIONS = [25, 50, 100, 200];

export default function Rules() {
  const [rules, setRules] = useState([]);
  const [clients, setClients] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [success, setSuccess] = useState(null);
  const [showForm, setShowForm] = useState(false);
  const [editingRule, setEditingRule] = useState(null);
  const [filterClientId, setFilterClientId] = useState('');
  const [formData, setFormData] = useState({
    client_id: '',
    severity: 'LOW',
    source: '',
    name: '',
  });

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [totalCount, setTotalCount] = useState(0);

  const totalPages = Math.ceil(totalCount / pageSize);

  useEffect(() => {
    loadClients();
  }, []);

  const loadClients = async () => {
    try {
      // Load only 100 clients for dropdown - sufficient for most use cases
      // and reduces database load from COUNT(*) queries
      const data = await clientsAPI.list(100, 0);
      setClients(data?.clients || []);
    } catch (err) {
      console.error('Failed to load clients:', err);
    }
  };

  const loadRules = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const clientId = filterClientId || null;
      const offset = (currentPage - 1) * pageSize;
      console.log('Loading rules, clientId:', clientId, 'limit:', pageSize, 'offset:', offset);
      const data = await rulesAPI.list(clientId, pageSize, offset);
      console.log('Rules API response:', data);
      
      if (data && data.rules) {
        setRules(data.rules || []);
        setTotalCount(data.total || 0);
        console.log(`Loaded ${data.rules?.length || 0} of ${data.total} rules`);
      } else if (Array.isArray(data)) {
        // Fallback for old API format
        setRules(data);
        setTotalCount(data.length);
      } else {
        console.warn('Unexpected response format:', typeof data, data);
        setRules([]);
        setTotalCount(0);
      }
    } catch (err) {
      console.error('Error loading rules:', err);
      setError(`Failed to load rules: ${err.message}`);
      setRules([]);
      setTotalCount(0);
    } finally {
      setLoading(false);
    }
  }, [filterClientId, currentPage, pageSize]);

  useEffect(() => {
    loadRules();
  }, [loadRules]);

  // Reset to page 1 when filters or page size change
  useEffect(() => {
    setCurrentPage(1);
  }, [filterClientId, pageSize]);

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

    // Validate form
    if (!formData.client_id) {
      setError('Please select a client');
      return;
    }
    if (!formData.source || !formData.name) {
      setError('Source and Name are required');
      return;
    }

    try {
      console.log('Creating rule:', formData);
      const result = await rulesAPI.create(
        formData.client_id,
        formData.severity,
        formData.source,
        formData.name
      );
      console.log('Rule created:', result);
      setSuccess('Rule created successfully!');
      resetForm();
      loadRules();
    } catch (err) {
      console.error('Error creating rule:', err);
      setError(`Failed to create rule: ${err.message}`);
    }
  };

  const handleUpdate = async (e) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    try {
      await rulesAPI.update(
        editingRule.rule_id,
        formData.severity,
        formData.source,
        formData.name,
        editingRule.version
      );
      setSuccess('Rule updated successfully!');
      resetForm();
      loadRules();
    } catch (err) {
      setError(err.message);
    }
  };

  const handleToggle = async (rule) => {
    setError(null);
    setSuccess(null);

    try {
      await rulesAPI.toggle(rule.rule_id, !rule.enabled, rule.version);
      setSuccess(`Rule ${rule.enabled ? 'disabled' : 'enabled'} successfully!`);
      loadRules();
    } catch (err) {
      setError(err.message);
    }
  };

  const handleDelete = async (ruleId) => {
    if (!confirm('Are you sure you want to delete this rule?')) {
      return;
    }

    setError(null);
    setSuccess(null);

    try {
      await rulesAPI.delete(ruleId);
      setSuccess('Rule deleted successfully!');
      loadRules();
    } catch (err) {
      setError(err.message);
    }
  };

  const startEdit = (rule) => {
    setEditingRule(rule);
    setFormData({
      client_id: rule.client_id,
      severity: rule.severity,
      source: rule.source,
      name: rule.name,
    });
    setShowForm(true);
  };

  const resetForm = () => {
    setFormData({ client_id: '', severity: 'LOW', source: '', name: '' });
    setEditingRule(null);
    setShowForm(false);
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <div className="card">
      <h2>Rules</h2>

      {error && <div className="error">Error: {error}</div>}
      {success && <div className="success">{success}</div>}

      <div style={{ marginBottom: '20px', display: 'flex', gap: '15px', flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <div className="form-group" style={{ maxWidth: '300px', marginBottom: 0 }}>
          <label>Filter by Client</label>
          <select
            value={filterClientId}
            onChange={(e) => setFilterClientId(e.target.value)}
          >
            <option value="">All Clients</option>
            {clients.map((client) => (
              <option key={client.client_id} value={client.client_id}>
                {client.name} ({client.client_id})
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
        <button className="btn btn-primary" onClick={() => {
          resetForm();
          setShowForm(true);
        }}>
          {showForm ? 'Cancel' : 'Create New Rule'}
        </button>
        <button className="btn btn-secondary" onClick={loadRules}>
          Refresh
        </button>
      </div>

      {/* Pagination info */}
      {totalCount > 0 && (
        <div style={{ marginBottom: '15px', color: '#666', fontSize: '14px' }}>
          Showing {((currentPage - 1) * pageSize) + 1} - {Math.min(currentPage * pageSize, totalCount)} of {totalCount} rules
        </div>
      )}

      {showForm && (
        <form onSubmit={editingRule ? handleUpdate : handleCreate} style={{ marginTop: '20px' }}>
          {!editingRule && (
            <div className="form-group">
              <label>Client *</label>
              <select
                value={formData.client_id}
                onChange={(e) => setFormData({ ...formData, client_id: e.target.value })}
                required
              >
                <option value="">Select a client</option>
                {clients.map((client) => (
                  <option key={client.client_id} value={client.client_id}>
                    {client.name} ({client.client_id})
                  </option>
                ))}
              </select>
            </div>
          )}
          <div className="form-group">
            <label>Severity *</label>
            <select
              value={formData.severity}
              onChange={(e) => setFormData({ ...formData, severity: e.target.value })}
              required
            >
              {SEVERITY_OPTIONS.map((sev) => (
                <option key={sev} value={sev}>
                  {sev}
                </option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label>Source *</label>
            <input
              type="text"
              value={formData.source}
              onChange={(e) => setFormData({ ...formData, source: e.target.value })}
              required
              placeholder="e.g., api, db, monitor"
            />
          </div>
          <div className="form-group">
            <label>Name *</label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
              placeholder="e.g., timeout, error, latency"
            />
          </div>
          <div className="button-group">
            <button type="submit" className="btn btn-primary">
              {editingRule ? 'Update Rule' : 'Create Rule'}
            </button>
            {editingRule && (
              <button type="button" className="btn btn-secondary" onClick={resetForm}>
                Cancel
              </button>
            )}
          </div>
        </form>
      )}

      {loading ? (
        <div className="loading">Loading rules...</div>
      ) : rules.length === 0 ? (
        <div className="empty-state">
          <p>No rules found. Create one to get started.</p>
        </div>
      ) : (
        <>
          <div style={{ overflowX: 'auto' }}>
            <table className="table">
              <thead>
                <tr>
                  <th>Rule ID</th>
                  <th>Client ID</th>
                  <th>Severity</th>
                  <th>Source</th>
                  <th>Name</th>
                  <th>Enabled</th>
                  <th>Version</th>
                  <th>Created At</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {rules.map((rule) => (
                  <tr key={rule.rule_id}>
                    <td style={{ fontSize: '12px' }}>{rule.rule_id}</td>
                    <td>{rule.client_id}</td>
                    <td>
                      <span className={`badge badge-${rule.severity === 'CRITICAL' ? 'danger' : 'success'}`}>
                        {rule.severity}
                      </span>
                    </td>
                    <td>{rule.source}</td>
                    <td>{rule.name}</td>
                    <td>
                      <span className={rule.enabled ? 'badge badge-success' : 'badge badge-danger'}>
                        {rule.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>
                    <td>{rule.version}</td>
                    <td>{formatDate(rule.created_at)}</td>
                    <td>
                      <div style={{ display: 'flex', gap: '5px' }}>
                        <button
                          className="btn btn-small btn-primary"
                          onClick={() => startEdit(rule)}
                        >
                          Edit
                        </button>
                        <button
                          className="btn btn-small btn-warning"
                          onClick={() => handleToggle(rule)}
                        >
                          {rule.enabled ? 'Disable' : 'Enable'}
                        </button>
                        <button
                          className="btn btn-small btn-danger"
                          onClick={() => handleDelete(rule.rule_id)}
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
