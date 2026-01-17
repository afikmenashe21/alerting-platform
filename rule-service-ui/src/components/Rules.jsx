import { useState, useEffect } from 'react';
import { rulesAPI, clientsAPI } from '../services/api';

const SEVERITY_OPTIONS = ['LOW', 'MEDIUM', 'HIGH', 'CRITICAL'];

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

  useEffect(() => {
    loadClients();
    loadRules();
  }, []);

  useEffect(() => {
    loadRules();
  }, [filterClientId]);

  const loadClients = async () => {
    try {
      const data = await clientsAPI.list();
      setClients(data || []);
    } catch (err) {
      console.error('Failed to load clients:', err);
    }
  };

  const loadRules = async () => {
    setLoading(true);
    setError(null);
    try {
      const clientId = filterClientId || null;
      console.log('Loading rules, clientId:', clientId);
      const data = await rulesAPI.list(clientId);
      console.log('Rules API response:', data);
      if (Array.isArray(data)) {
        setRules(data);
        console.log(`Loaded ${data.length} rules`);
      } else {
        console.warn('Expected array but got:', typeof data, data);
        setRules([]);
      }
    } catch (err) {
      console.error('Error loading rules:', err);
      setError(`Failed to load rules: ${err.message}`);
      setRules([]);
    } finally {
      setLoading(false);
    }
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

      <div style={{ marginBottom: '20px' }}>
        <div className="form-group" style={{ maxWidth: '300px' }}>
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
      </div>

      <div className="button-group">
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
      )}
    </div>
  );
}
