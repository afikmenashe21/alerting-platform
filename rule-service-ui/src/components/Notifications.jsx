import { useState, useEffect, useCallback } from 'react';
import { notificationsAPI, clientsAPI } from '../services/api';

const STATUS_OPTIONS = ['RECEIVED', 'SENT', 'FAILED'];
const PAGE_SIZE_OPTIONS = [25, 50, 100, 200];

export default function Notifications() {
  const [notifications, setNotifications] = useState([]);
  const [clients, setClients] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [filterClientId, setFilterClientId] = useState('');
  const [filterStatus, setFilterStatus] = useState('');
  
  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [totalCount, setTotalCount] = useState(0);

  const totalPages = Math.ceil(totalCount / pageSize);

  useEffect(() => {
    loadClients();
  }, []);

  const loadNotifications = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const clientId = filterClientId || null;
      const status = filterStatus || null;
      const offset = (currentPage - 1) * pageSize;
      
      console.log('Loading notifications, clientId:', clientId, 'status:', status, 'limit:', pageSize, 'offset:', offset);
      const data = await notificationsAPI.list(clientId, status, pageSize, offset);
      console.log('Notifications API response:', data);
      
      if (data && data.notifications) {
        setNotifications(data.notifications || []);
        setTotalCount(data.total || 0);
        console.log(`Loaded ${data.notifications?.length || 0} of ${data.total} notifications`);
      } else if (Array.isArray(data)) {
        // Fallback for old API format
        setNotifications(data);
        setTotalCount(data.length);
      } else {
        console.warn('Unexpected response format:', typeof data, data);
        setNotifications([]);
        setTotalCount(0);
      }
    } catch (err) {
      console.error('Error loading notifications:', err);
      setError(`Failed to load notifications: ${err.message}`);
      setNotifications([]);
      setTotalCount(0);
    } finally {
      setLoading(false);
    }
  }, [filterClientId, filterStatus, currentPage, pageSize]);

  useEffect(() => {
    loadNotifications();
  }, [loadNotifications]);

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [filterClientId, filterStatus, pageSize]);

  const loadClients = async () => {
    try {
      const data = await clientsAPI.list(200, 0);
      setClients(data?.clients || []);
    } catch (err) {
      console.error('Failed to load clients:', err);
    }
  };

  const handlePageChange = (newPage) => {
    if (newPage >= 1 && newPage <= totalPages) {
      setCurrentPage(newPage);
    }
  };

  const handlePageSizeChange = (newSize) => {
    setPageSize(newSize);
    setCurrentPage(1);
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

  return (
    <div className="card">
      <h2>Notifications</h2>

      {error && <div className="error">Error: {error}</div>}

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
        <button className="btn btn-secondary" onClick={loadNotifications}>
          Refresh
        </button>
      </div>

      {/* Pagination info */}
      {totalCount > 0 && (
        <div style={{ marginBottom: '15px', color: '#666', fontSize: '14px' }}>
          Showing {((currentPage - 1) * pageSize) + 1} - {Math.min(currentPage * pageSize, totalCount)} of {totalCount} notifications
        </div>
      )}

      {loading ? (
        <div className="loading">Loading notifications...</div>
      ) : notifications.length === 0 ? (
        <div className="empty-state">
          <p>No notifications found.</p>
        </div>
      ) : (
        <>
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
                      <span className={`badge ${notification.status === 'SENT' ? 'badge-success' : notification.status === 'FAILED' ? 'badge-danger' : 'badge-warning'}`}>
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
