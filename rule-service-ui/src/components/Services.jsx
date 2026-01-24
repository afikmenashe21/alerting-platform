import { useState, useEffect, useCallback } from 'react';
import { serviceMetricsAPI } from '../services/api';

function Services() {
  const [metrics, setMetrics] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [activeService, setActiveService] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);

  const fetchMetrics = useCallback(async () => {
    try {
      const data = await serviceMetricsAPI.getAll();
      setMetrics(data);
      setLastUpdated(new Date());
      setError(null);
      
      // Set first service as active if none selected
      if (!activeService && data.known_services?.length > 0) {
        setActiveService(data.known_services[0]);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [activeService]);

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 15000); // Refresh every 15s
    return () => clearInterval(interval);
  }, [fetchMetrics]);

  if (loading && !metrics) {
    return <div className="loading">Loading service metrics...</div>;
  }

  if (error && !metrics) {
    return <div className="error">Error loading service metrics: {error}</div>;
  }

  const getStatusColor = (status) => {
    switch (status) {
      case 'healthy': return '#10b981';
      case 'unhealthy': return '#f59e0b';
      case 'offline': return '#ef4444';
      default: return '#6b7280';
    }
  };

  const getStatusIcon = (status) => {
    switch (status) {
      case 'healthy': return '●';
      case 'unhealthy': return '◐';
      case 'offline': return '○';
      default: return '?';
    }
  };

  const formatUptime = (startedAt) => {
    if (!startedAt) return 'N/A';
    const started = new Date(startedAt);
    const now = new Date();
    const diff = now - started;
    
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    
    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
  };

  const formatNumber = (num) => {
    if (num === undefined || num === null) return '0';
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
  };

  const activeMetrics = activeService ? metrics?.services?.[activeService] : null;

  return (
    <div className="services-dashboard">
      <div className="services-header">
        <h2>Service Health & Metrics</h2>
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

      {/* Service Status Overview */}
      <div className="services-overview">
        {metrics?.known_services?.map((serviceName) => {
          const service = metrics.services?.[serviceName] || { status: 'offline' };
          return (
            <div
              key={serviceName}
              className={`service-card ${activeService === serviceName ? 'active' : ''}`}
              onClick={() => setActiveService(serviceName)}
            >
              <div className="service-status" style={{ color: getStatusColor(service.status) }}>
                {getStatusIcon(service.status)}
              </div>
              <div className="service-info">
                <div className="service-name">{serviceName}</div>
                <div className="service-status-text">{service.status || 'offline'}</div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Selected Service Details */}
      {activeService && (
        <div className="service-details">
          <h3>{activeService}</h3>
          
          {activeMetrics?.status === 'offline' ? (
            <div className="offline-message">
              <span className="offline-icon">⚠️</span>
              <div>
                <strong>Service Offline</strong>
                <p>This service is not reporting metrics. It may be stopped or not yet deployed.</p>
              </div>
            </div>
          ) : (
            <>
              {/* Key Metrics */}
              <div className="metrics-grid">
                <div className="metric-box">
                  <div className="metric-label">Status</div>
                  <div className="metric-value" style={{ color: getStatusColor(activeMetrics?.status) }}>
                    {activeMetrics?.status || 'unknown'}
                  </div>
                </div>
                <div className="metric-box">
                  <div className="metric-label">Uptime</div>
                  <div className="metric-value">{formatUptime(activeMetrics?.started_at)}</div>
                </div>
                <div className="metric-box">
                  <div className="metric-label">Messages/sec</div>
                  <div className="metric-value">
                    {activeMetrics?.messages_per_second?.toFixed(2) || '0.00'}
                  </div>
                </div>
                <div className="metric-box">
                  <div className="metric-label">Avg Latency</div>
                  <div className="metric-value">
                    {activeMetrics?.avg_processing_latency_ns?.toFixed(0) || '0'} ns
                  </div>
                </div>
              </div>

              {/* Message Counters */}
              <div className="counters-section">
                <h4>Message Counters (since start)</h4>
                <div className="counters-grid">
                  <div className="counter">
                    <span className="counter-label">Received</span>
                    <span className="counter-value">{formatNumber(activeMetrics?.messages_received)}</span>
                  </div>
                  <div className="counter">
                    <span className="counter-label">Processed</span>
                    <span className="counter-value">{formatNumber(activeMetrics?.messages_processed)}</span>
                  </div>
                  <div className="counter">
                    <span className="counter-label">Published</span>
                    <span className="counter-value">{formatNumber(activeMetrics?.messages_published)}</span>
                  </div>
                  <div className="counter error">
                    <span className="counter-label">Errors</span>
                    <span className="counter-value">{formatNumber(activeMetrics?.processing_errors)}</span>
                  </div>
                </div>
              </div>

              {/* Custom Counters */}
              {activeMetrics?.custom_counters && Object.keys(activeMetrics.custom_counters).length > 0 && (
                <div className="custom-counters-section">
                  <h4>Service-Specific Metrics</h4>
                  <div className="custom-counters">
                    {Object.entries(activeMetrics.custom_counters).map(([name, value]) => (
                      <div key={name} className="custom-counter">
                        <span className="counter-label">{name.replace(/_/g, ' ')}</span>
                        <span className="counter-value">{formatNumber(value)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Last Updated */}
              {activeMetrics?.last_updated && (
                <div className="last-report">
                  Last report: {new Date(activeMetrics.last_updated).toLocaleString()}
                </div>
              )}
            </>
          )}
        </div>
      )}

      <style>{`
        .services-dashboard {
          padding: 20px 0;
        }
        
        .services-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 24px;
        }
        
        .services-header h2 {
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
        
        .services-overview {
          display: flex;
          flex-wrap: wrap;
          gap: 12px;
          margin-bottom: 24px;
        }
        
        .service-card {
          display: flex;
          align-items: center;
          gap: 12px;
          background: #f8fafc;
          border: 2px solid #e2e8f0;
          border-radius: 12px;
          padding: 16px 20px;
          cursor: pointer;
          transition: all 0.2s ease;
          min-width: 180px;
        }
        
        .service-card:hover {
          border-color: #cbd5e1;
          background: #f1f5f9;
        }
        
        .service-card.active {
          border-color: #3b82f6;
          background: #eff6ff;
        }
        
        .service-status {
          font-size: 24px;
          line-height: 1;
        }
        
        .service-info {
          display: flex;
          flex-direction: column;
        }
        
        .service-name {
          font-weight: 600;
          color: #1e293b;
          font-size: 14px;
        }
        
        .service-status-text {
          font-size: 12px;
          color: #64748b;
          text-transform: capitalize;
        }
        
        .service-details {
          background: #f8fafc;
          border: 1px solid #e2e8f0;
          border-radius: 12px;
          padding: 24px;
        }
        
        .service-details h3 {
          margin: 0 0 20px 0;
          color: #1e293b;
        }
        
        .offline-message {
          display: flex;
          align-items: flex-start;
          gap: 16px;
          padding: 20px;
          background: #fef3c7;
          border-radius: 8px;
        }
        
        .offline-icon {
          font-size: 24px;
        }
        
        .offline-message strong {
          display: block;
          margin-bottom: 4px;
          color: #92400e;
        }
        
        .offline-message p {
          margin: 0;
          color: #a16207;
          font-size: 14px;
        }
        
        .metrics-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
          gap: 16px;
          margin-bottom: 24px;
        }
        
        .metric-box {
          background: white;
          border: 1px solid #e2e8f0;
          border-radius: 8px;
          padding: 16px;
          text-align: center;
        }
        
        .metric-label {
          font-size: 12px;
          color: #64748b;
          text-transform: uppercase;
          letter-spacing: 0.5px;
          margin-bottom: 8px;
        }
        
        .metric-value {
          font-size: 24px;
          font-weight: 700;
          color: #1e293b;
        }
        
        .counters-section, .custom-counters-section {
          margin-bottom: 20px;
        }
        
        .counters-section h4, .custom-counters-section h4 {
          margin: 0 0 12px 0;
          font-size: 14px;
          color: #475569;
        }
        
        .counters-grid, .custom-counters {
          display: flex;
          flex-wrap: wrap;
          gap: 12px;
        }
        
        .counter, .custom-counter {
          display: flex;
          justify-content: space-between;
          align-items: center;
          background: white;
          border: 1px solid #e2e8f0;
          border-radius: 8px;
          padding: 12px 16px;
          min-width: 160px;
        }
        
        .counter.error {
          border-color: #fecaca;
          background: #fef2f2;
        }
        
        .counter-label {
          font-size: 13px;
          color: #64748b;
          text-transform: capitalize;
        }
        
        .counter-value {
          font-size: 18px;
          font-weight: 600;
          color: #1e293b;
        }
        
        .counter.error .counter-value {
          color: #dc2626;
        }
        
        .last-report {
          margin-top: 16px;
          padding-top: 16px;
          border-top: 1px solid #e2e8f0;
          font-size: 12px;
          color: #94a3b8;
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

export default Services;
