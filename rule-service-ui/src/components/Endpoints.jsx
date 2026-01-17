import { useState, useEffect } from 'react';
import { endpointsAPI, rulesAPI } from '../services/api';

const ENDPOINT_TYPES = ['email', 'webhook', 'slack'];

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

  useEffect(() => {
    loadRules();
    loadEndpoints();
  }, []);

  useEffect(() => {
    loadEndpoints();
  }, [filterRuleId]);

  const loadRules = async () => {
    try {
      const data = await rulesAPI.list();
      setRules(data || []);
    } catch (err) {
      console.error('Failed to load rules:', err);
    }
  };

  const loadEndpoints = async () => {
    if (!filterRuleId) {
      // If no filter, we need to load endpoints for all rules
      // For simplicity, we'll show a message to select a rule
      setEndpoints([]);
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const data = await endpointsAPI.list(filterRuleId);
      console.log('Endpoints API response:', data);
      if (Array.isArray(data)) {
        setEndpoints(data);
      } else {
        console.warn('Expected array but got:', typeof data, data);
        setEndpoints([]);
      }
    } catch (err) {
      console.error('Error loading endpoints:', err);
      setError(err.message);
      setEndpoints([]);
    } finally {
      setLoading(false);
    }
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

      <div style={{ marginBottom: '20px' }}>
        <div className="form-group" style={{ maxWidth: '400px' }}>
          <label>Filter by Rule *</label>
          <select
            value={filterRuleId}
            onChange={(e) => {
              setFilterRuleId(e.target.value);
              setFormData({ ...formData, rule_id: e.target.value });
            }}
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
      </div>

      {filterRuleId && (
        <>
          <div className="button-group">
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
              <p>No endpoints found for this rule. Create one to get started.</p>
            </div>
          ) : (
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
          )}
        </>
      )}

      {!filterRuleId && (
        <div className="empty-state">
          <p>Please select a rule to view and manage its endpoints.</p>
        </div>
      )}
    </div>
  );
}
