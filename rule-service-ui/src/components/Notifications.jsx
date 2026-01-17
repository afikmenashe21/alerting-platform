import { useState, useEffect } from 'react';
import { notificationsAPI, clientsAPI } from '../services/api';

const STATUS_OPTIONS = ['RECEIVED', 'SENT'];

export default function Notifications() {
  const [notifications, setNotifications] = useState([]);
  const [clients, setClients] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [filterClientId, setFilterClientId] = useState('');
  const [filterStatus, setFilterStatus] = useState('');

  useEffect(() => {
    loadClients();
    loadNotifications();
  }, []);

  useEffect(() => {
    loadNotifications();
  }, [filterClientId, filterStatus]);

  const loadClients = async () => {
    try {
      const data = await clientsAPI.list();
      setClients(data || []);
    } catch (err) {
      console.error('Failed to load clients:', err);
    }
  };

  const loadNotifications = async () => {
    setLoading(true);
    setError(null);
    try {
      const clientId = filterClientId || null;
      const status = filterStatus || null;
      console.log('Loading notifications, clientId:', clientId, 'status:', status);
      const data = await notificationsAPI.list(clientId, status);
      console.log('Notifications API response:', data);
      if (Array.isArray(data)) {
        setNotifications(data);
        console.log(`Loaded ${data.length} notifications`);
      } else {
        console.warn('Expected array but got:', typeof data, data);
        setNotifications([]);
      }
    } catch (err) {
      console.error('Error loading notifications:', err);
      setError(`Failed to load notifications: ${err.message}`);
      setNotifications([]);
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  const formatContext = (context) => {
    if (!context || Object.keys(context).length === 0) {
      return '-';
    }
    return JSON.stringify(context);
  };

  const formatRuleIDs = (ruleIDs) => {
    if (!ruleIDs || ruleIDs.length === 0) {
      return '-';
    }
    return ruleIDs.join(', ');
  };

  return (
    <div className="card">
      <h2>Notifications</h2>

      {error && <div className="error">Error: {error}</div>}

      <div style={{ marginBottom: '20px', display: 'flex', gap: '15px', flexWrap: 'wrap' }}>
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
        <div className="form-group" style={{ maxWidth: '200px', marginBottom: 0 }}>
          <label>Filter by Status</label>
          <select
            value={filterStatus}
            onChange={(e) => setFilterStatus(e.target.value)}
          >
            <option value="">All Statuses</option>
            {STATUS_OPTIONS.map((status) => (
              <option key={status} value={status}>
                {status}
              </option>
            ))}
          </select>
        </div>
        <div style={{ display: 'flex', alignItems: 'flex-end' }}>
          <button className="btn btn-secondary" onClick={loadNotifications}>
            Refresh
          </button>
        </div>
      </div>

      {loading ? (
        <div className="loading">Loading notifications...</div>
      ) : notifications.length === 0 ? (
        <div className="empty-state">
          <p>No notifications found.</p>
        </div>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table className="table">
            <thead>
              <tr>
                <th>Notification ID</th>
                <th>Client ID</th>
                <th>Alert ID</th>
                <th>Severity</th>
                <th>Source</th>
                <th>Name</th>
                <th>Status</th>
                <th>Rule IDs</th>
                <th>Context</th>
                <th>Created At</th>
                <th>Updated At</th>
              </tr>
            </thead>
            <tbody>
              {notifications.map((notification) => (
                <tr key={notification.notification_id}>
                  <td style={{ fontSize: '12px' }}>{notification.notification_id}</td>
                  <td>{notification.client_id}</td>
                  <td style={{ fontSize: '12px' }}>{notification.alert_id}</td>
                  <td>
                    <span className={`badge badge-${notification.severity === 'CRITICAL' || notification.severity === 'HIGH' ? 'danger' : 'success'}`}>
                      {notification.severity}
                    </span>
                  </td>
                  <td>{notification.source}</td>
                  <td>{notification.name}</td>
                  <td>
                    <span className={notification.status === 'SENT' ? 'badge badge-success' : 'badge badge-warning'}>
                      {notification.status}
                    </span>
                  </td>
                  <td style={{ fontSize: '11px', maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {formatRuleIDs(notification.rule_ids)}
                  </td>
                  <td style={{ fontSize: '11px', maxWidth: '150px', overflow: 'hidden', textOverflow: 'ellipsis' }} title={formatContext(notification.context)}>
                    {formatContext(notification.context)}
                  </td>
                  <td>{formatDate(notification.created_at)}</td>
                  <td>{formatDate(notification.updated_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
