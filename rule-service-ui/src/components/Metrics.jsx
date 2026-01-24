import { useState, useEffect, useCallback } from 'react';
import { metricsAPI } from '../services/api';

function Metrics() {
  const [metrics, setMetrics] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);

  const fetchMetrics = useCallback(async () => {
    try {
      const data = await metricsAPI.get();
      setMetrics(data);
      setLastUpdated(new Date());
      setError(null);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchMetrics();
    // Auto-refresh every 30 seconds
    const interval = setInterval(fetchMetrics, 30000);
    return () => clearInterval(interval);
  }, [fetchMetrics]);

  if (loading && !metrics) {
    return <div className="loading">Loading metrics...</div>;
  }

  if (error && !metrics) {
    return <div className="error">Error loading metrics: {error}</div>;
  }

  const getStatusColor = (status) => {
    switch (status) {
      case 'SENT': return '#10b981'; // green
      case 'PENDING': return '#f59e0b'; // amber
      case 'FAILED': return '#ef4444'; // red
      default: return '#6b7280'; // gray
    }
  };

  const formatNumber = (num) => {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num?.toString() || '0';
  };

  // Calculate max for chart scaling
  const maxHourlyCount = metrics?.notifications_by_hour?.length > 0 
    ? Math.max(...metrics.notifications_by_hour.map(h => h.count), 1)
    : 1;

  return (
    <div className="metrics-dashboard">
      <div className="metrics-header">
        <h2>System Metrics Dashboard</h2>
        <div className="refresh-info">
          {lastUpdated && (
            <span className="last-updated">
              Last updated: {lastUpdated.toLocaleTimeString()}
            </span>
          )}
          <button onClick={fetchMetrics} className="refresh-btn" disabled={loading}>
            {loading ? 'Refreshing...' : 'Refresh'}
          </button>
        </div>
      </div>

      {error && <div className="error-banner">Failed to refresh: {error}</div>}

      {/* Summary Cards */}
      <div className="metrics-cards">
        <div className="metric-card">
          <div className="metric-value">{formatNumber(metrics?.total_notifications)}</div>
          <div className="metric-label">Total Notifications</div>
          <div className="metric-detail">
            Last hour: {formatNumber(metrics?.notifications_last_hour)} | 
            Last 24h: {formatNumber(metrics?.notifications_last_24h)}
          </div>
        </div>

        <div className="metric-card">
          <div className="metric-value">{formatNumber(metrics?.total_rules)}</div>
          <div className="metric-label">Total Rules</div>
          <div className="metric-detail">
            <span style={{color: '#10b981'}}>{metrics?.enabled_rules} enabled</span> | 
            <span style={{color: '#6b7280'}}> {metrics?.disabled_rules} disabled</span>
          </div>
        </div>

        <div className="metric-card">
          <div className="metric-value">{formatNumber(metrics?.total_clients)}</div>
          <div className="metric-label">Clients</div>
        </div>

        <div className="metric-card">
          <div className="metric-value">{formatNumber(metrics?.total_endpoints)}</div>
          <div className="metric-label">Endpoints</div>
          <div className="metric-detail">
            {metrics?.enabled_endpoints} active
          </div>
        </div>
      </div>

      {/* Notification Status Breakdown */}
      <div className="metrics-section">
        <h3>Notifications by Status</h3>
        <div className="status-bars">
          {Object.entries(metrics?.notifications_by_status || {}).map(([status, count]) => (
            <div key={status} className="status-bar-container">
              <div className="status-bar-label">
                <span className="status-name">{status}</span>
                <span className="status-count">{formatNumber(count)}</span>
              </div>
              <div className="status-bar-bg">
                <div 
                  className="status-bar-fill" 
                  style={{
                    width: `${(count / (metrics?.total_notifications || 1)) * 100}%`,
                    backgroundColor: getStatusColor(status)
                  }}
                />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Endpoints by Type */}
      <div className="metrics-section">
        <h3>Endpoints by Type</h3>
        <div className="endpoint-types">
          {Object.entries(metrics?.endpoints_by_type || {}).map(([type, count]) => (
            <div key={type} className="endpoint-type-badge">
              <span className="type-icon">
                {type === 'email' ? '📧' : type === 'slack' ? '💬' : type === 'webhook' ? '🔗' : '📤'}
              </span>
              <span className="type-name">{type}</span>
              <span className="type-count">{count}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Hourly Chart */}
      {metrics?.notifications_by_hour?.length > 0 && (
        <div className="metrics-section">
          <h3>Notifications (Last 24 Hours)</h3>
          <div className="hourly-chart">
            {metrics.notifications_by_hour.map((hourData, index) => {
              const hour = new Date(hourData.hour);
              const heightPercent = (hourData.count / maxHourlyCount) * 100;
              return (
                <div key={index} className="chart-bar-container" title={`${hour.toLocaleString()}: ${hourData.count}`}>
                  <div className="chart-bar-wrapper">
                    <div 
                      className="chart-bar" 
                      style={{ height: `${Math.max(heightPercent, 2)}%` }}
                    />
                  </div>
                  <span className="chart-label">
                    {hour.getHours().toString().padStart(2, '0')}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      <style>{`
        .metrics-dashboard {
          padding: 20px 0;
        }
        
        .metrics-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 24px;
        }
        
        .metrics-header h2 {
          margin: 0;
        }
        
        .refresh-info {
          display: flex;
          align-items: center;
          gap: 12px;
        }
        
        .last-updated {
          color: #6b7280;
          font-size: 14px;
        }
        
        .refresh-btn {
          padding: 8px 16px;
          background: #3b82f6;
          color: white;
          border: none;
          border-radius: 6px;
          cursor: pointer;
          font-size: 14px;
        }
        
        .refresh-btn:hover {
          background: #2563eb;
        }
        
        .refresh-btn:disabled {
          background: #9ca3af;
          cursor: not-allowed;
        }
        
        .error-banner {
          background: #fef2f2;
          color: #dc2626;
          padding: 12px;
          border-radius: 8px;
          margin-bottom: 20px;
        }
        
        .metrics-cards {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
          gap: 20px;
          margin-bottom: 32px;
        }
        
        .metric-card {
          background: #f8fafc;
          border: 1px solid #e2e8f0;
          border-radius: 12px;
          padding: 20px;
          text-align: center;
        }
        
        .metric-value {
          font-size: 36px;
          font-weight: 700;
          color: #1e293b;
        }
        
        .metric-label {
          font-size: 14px;
          color: #64748b;
          margin-top: 4px;
        }
        
        .metric-detail {
          font-size: 12px;
          color: #94a3b8;
          margin-top: 8px;
        }
        
        .metrics-section {
          background: #f8fafc;
          border: 1px solid #e2e8f0;
          border-radius: 12px;
          padding: 20px;
          margin-bottom: 20px;
        }
        
        .metrics-section h3 {
          margin: 0 0 16px 0;
          font-size: 16px;
          color: #334155;
        }
        
        .status-bars {
          display: flex;
          flex-direction: column;
          gap: 12px;
        }
        
        .status-bar-container {
          width: 100%;
        }
        
        .status-bar-label {
          display: flex;
          justify-content: space-between;
          margin-bottom: 4px;
          font-size: 14px;
        }
        
        .status-name {
          font-weight: 500;
          color: #334155;
        }
        
        .status-count {
          color: #64748b;
        }
        
        .status-bar-bg {
          height: 8px;
          background: #e2e8f0;
          border-radius: 4px;
          overflow: hidden;
        }
        
        .status-bar-fill {
          height: 100%;
          border-radius: 4px;
          transition: width 0.3s ease;
        }
        
        .endpoint-types {
          display: flex;
          flex-wrap: wrap;
          gap: 12px;
        }
        
        .endpoint-type-badge {
          display: flex;
          align-items: center;
          gap: 8px;
          background: white;
          border: 1px solid #e2e8f0;
          border-radius: 8px;
          padding: 12px 16px;
        }
        
        .type-icon {
          font-size: 20px;
        }
        
        .type-name {
          font-weight: 500;
          color: #334155;
          text-transform: capitalize;
        }
        
        .type-count {
          background: #e2e8f0;
          color: #475569;
          padding: 2px 8px;
          border-radius: 12px;
          font-size: 12px;
          font-weight: 600;
        }
        
        .hourly-chart {
          display: flex;
          align-items: flex-end;
          gap: 4px;
          height: 150px;
          padding-top: 10px;
        }
        
        .chart-bar-container {
          flex: 1;
          display: flex;
          flex-direction: column;
          align-items: center;
          height: 100%;
        }
        
        .chart-bar-wrapper {
          flex: 1;
          width: 100%;
          display: flex;
          align-items: flex-end;
          justify-content: center;
        }
        
        .chart-bar {
          width: 80%;
          max-width: 30px;
          background: linear-gradient(to top, #3b82f6, #60a5fa);
          border-radius: 4px 4px 0 0;
          transition: height 0.3s ease;
        }
        
        .chart-label {
          font-size: 10px;
          color: #94a3b8;
          margin-top: 4px;
        }
        
        .loading {
          text-align: center;
          padding: 40px;
          color: #64748b;
        }
        
        .error {
          text-align: center;
          padding: 40px;
          color: #dc2626;
        }
      `}</style>
    </div>
  );
}

export default Metrics;
