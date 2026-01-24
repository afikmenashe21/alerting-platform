import { useState, useEffect, useCallback } from 'react';
import { alertGeneratorAPI } from '../services/api';

// Load test presets based on performance testing results
const PRESETS = {
  single: {
    label: 'Single Alert',
    description: 'Send 1 test alert',
    config: { single_test: true, severity: 'LOW', source: 'test', name: 'single-test' }
  },
  burst: {
    label: 'Max Burst (100K)',
    description: 'Send 100,000 alerts as fast as possible',
    config: { burst: 100000, test: true }
  },
  load: {
    label: 'Max Load (3 min)',
    description: '800 RPS for 3 minutes (~144K alerts)',
    config: { rps: 800, duration: '3m', test: true }
  }
};

function AlertGenerator() {
  const [activeJob, setActiveJob] = useState(null);
  const [jobHistory, setJobHistory] = useState([]);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);

  // Load job history and restore active job on mount
  useEffect(() => {
    loadJobHistory();
    const savedJobId = localStorage.getItem('alertGenerator_activeJobId');
    if (savedJobId) restoreActiveJob(savedJobId);
  }, []);

  // Poll active job status
  useEffect(() => {
    if (activeJob) {
      localStorage.setItem('alertGenerator_activeJobId', activeJob.id);
    } else {
      localStorage.removeItem('alertGenerator_activeJobId');
      return;
    }

    const interval = setInterval(async () => {
      try {
        const status = await alertGeneratorAPI.getStatus(activeJob.id);
        setActiveJob(status);
        if (['completed', 'failed', 'cancelled'].includes(status.status)) {
          clearInterval(interval);
          loadJobHistory();
          setActiveJob(null);
        }
      } catch (err) {
        if (err.message?.includes('not found')) {
          clearInterval(interval);
          setActiveJob(null);
        }
      }
    }, 500);

    return () => clearInterval(interval);
  }, [activeJob?.id]);

  const restoreActiveJob = async (jobId) => {
    try {
      const status = await alertGeneratorAPI.getStatus(jobId);
      if (['running', 'pending'].includes(status.status)) {
        setActiveJob(status);
      } else {
        localStorage.removeItem('alertGenerator_activeJobId');
      }
    } catch {
      localStorage.removeItem('alertGenerator_activeJobId');
    }
  };

  const loadJobHistory = useCallback(async () => {
    try {
      const jobs = await alertGeneratorAPI.list();
      setJobHistory(jobs.slice(0, 10));
    } catch (err) {
      if (jobHistory.length === 0) {
        setError('Cannot connect to alert-producer API (port 8082)');
      }
    }
  }, [jobHistory.length]);

  const startJob = async (preset) => {
    setError(null);
    setLoading(true);
    try {
      const response = await alertGeneratorAPI.generate(preset.config);
      const status = await alertGeneratorAPI.getStatus(response.job_id);
      setActiveJob(status);
      loadJobHistory();
    } catch (err) {
      setError(err.message?.includes('Failed to fetch') 
        ? 'Cannot connect to alert-producer API' 
        : err.message);
    } finally {
      setLoading(false);
    }
  };

  const stopJob = async () => {
    if (!activeJob) return;
    try {
      await alertGeneratorAPI.stop(activeJob.id);
      const status = await alertGeneratorAPI.getStatus(activeJob.id);
      setActiveJob(status);
      loadJobHistory();
    } catch (err) {
      setError(err.message);
    }
  };

  const formatTime = (timeStr) => timeStr ? new Date(timeStr).toLocaleString() : '-';
  
  const getStatusColor = (status) => ({
    completed: '#28a745',
    running: '#007bff',
    failed: '#dc3545',
    cancelled: '#6c757d'
  }[status] || '#6c757d');

  const getProgress = () => {
    if (!activeJob?.config) return null;
    const sent = activeJob.alerts_sent || 0;
    const { burst, rps, duration } = activeJob.config;
    
    if (burst > 0) {
      return { sent, total: burst, percent: Math.round((sent / burst) * 100) };
    }
    if (rps && duration) {
      const durationSec = parseInt(duration) * (duration.includes('m') ? 60 : 1);
      const total = Math.round(rps * durationSec);
      return { sent, total, percent: Math.min(Math.round((sent / total) * 100), 100) };
    }
    return { sent, total: null, percent: null };
  };

  return (
    <div className="alert-generator">
      <h2>Load Test Generator</h2>
      
      {error && (
        <div style={{ 
          color: '#dc3545', 
          marginBottom: '1rem', 
          padding: '0.75rem', 
          background: '#f8d7da', 
          borderRadius: '4px' 
        }}>
          {error}
        </div>
      )}

      {/* Preset Buttons */}
      <div style={{ marginBottom: '2rem' }}>
        <p style={{ color: '#6c757d', marginBottom: '1rem' }}>
          Select a load test mode. Results based on t3.small (~780 alerts/sec max).
        </p>
        <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
          {Object.entries(PRESETS).map(([key, preset]) => (
            <button
              key={key}
              onClick={() => startJob(preset)}
              disabled={loading || activeJob}
              style={{
                padding: '1rem 1.5rem',
                fontSize: '1rem',
                cursor: loading || activeJob ? 'not-allowed' : 'pointer',
                background: loading || activeJob ? '#e9ecef' : 
                  key === 'single' ? '#17a2b8' : 
                  key === 'burst' ? '#ffc107' : '#dc3545',
                color: key === 'burst' ? '#212529' : 'white',
                border: 'none',
                borderRadius: '8px',
                minWidth: '200px',
                textAlign: 'left',
                opacity: loading || activeJob ? 0.6 : 1
              }}
            >
              <div style={{ fontWeight: 'bold', marginBottom: '0.25rem' }}>
                {preset.label}
              </div>
              <div style={{ fontSize: '0.85rem', opacity: 0.9 }}>
                {preset.description}
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Active Job Status */}
      {activeJob && (
        <div style={{ 
          marginBottom: '2rem', 
          padding: '1.5rem', 
          border: '2px solid #007bff', 
          borderRadius: '8px', 
          background: '#f8f9fa' 
        }}>
          <div style={{ 
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center', 
            marginBottom: '1rem' 
          }}>
            <h3 style={{ margin: 0 }}>
              Active Job
              <span style={{ 
                color: getStatusColor(activeJob.status), 
                marginLeft: '0.75rem',
                fontSize: '0.9rem'
              }}>
                {activeJob.status.toUpperCase()}
              </span>
            </h3>
            {activeJob.status === 'running' && (
              <button
                onClick={stopJob}
                style={{
                  padding: '0.5rem 1rem',
                  background: '#dc3545',
                  color: 'white',
                  border: 'none',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  fontWeight: 'bold'
                }}
              >
                Stop
              </button>
            )}
          </div>

          {/* Progress Bar */}
          {activeJob.status === 'running' && (() => {
            const progress = getProgress();
            if (!progress) return null;
            
            return (
              <div style={{ marginBottom: '1rem' }}>
                <div style={{ 
                  display: 'flex', 
                  justifyContent: 'space-between', 
                  marginBottom: '0.5rem' 
                }}>
                  <span>
                    <strong>{progress.sent.toLocaleString()}</strong>
                    {progress.total && <> / {progress.total.toLocaleString()}</>} alerts
                  </span>
                  {progress.percent !== null && (
                    <strong>{progress.percent}%</strong>
                  )}
                </div>
                {progress.percent !== null && (
                  <div style={{
                    width: '100%',
                    height: '24px',
                    background: '#e9ecef',
                    borderRadius: '12px',
                    overflow: 'hidden'
                  }}>
                    <div style={{
                      width: `${progress.percent}%`,
                      height: '100%',
                      background: 'linear-gradient(90deg, #28a745, #20c997)',
                      transition: 'width 0.3s ease'
                    }} />
                  </div>
                )}
              </div>
            );
          })()}

          <div style={{ 
            display: 'grid', 
            gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', 
            gap: '0.5rem',
            fontSize: '0.9rem'
          }}>
            <div><strong>Started:</strong> {formatTime(activeJob.started_at)}</div>
            {activeJob.completed_at && (
              <div><strong>Completed:</strong> {formatTime(activeJob.completed_at)}</div>
            )}
          </div>
          
          {activeJob.error && (
            <div style={{ 
              marginTop: '0.75rem', 
              padding: '0.5rem', 
              background: '#f8d7da', 
              borderRadius: '4px', 
              color: '#dc3545' 
            }}>
              {activeJob.error}
            </div>
          )}
        </div>
      )}

      {/* Job History */}
      {jobHistory.length > 0 && (
        <div>
          <h3>Recent Jobs</h3>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9rem' }}>
            <thead>
              <tr style={{ borderBottom: '2px solid #ddd', textAlign: 'left' }}>
                <th style={{ padding: '0.5rem' }}>Status</th>
                <th style={{ padding: '0.5rem' }}>Alerts</th>
                <th style={{ padding: '0.5rem' }}>Started</th>
                <th style={{ padding: '0.5rem' }}>Completed</th>
              </tr>
            </thead>
            <tbody>
              {jobHistory.map((job) => (
                <tr key={job.id} style={{ borderBottom: '1px solid #eee' }}>
                  <td style={{ padding: '0.5rem' }}>
                    <span style={{ 
                      color: getStatusColor(job.status),
                      fontWeight: 'bold'
                    }}>
                      {job.status.toUpperCase()}
                    </span>
                  </td>
                  <td style={{ padding: '0.5rem' }}>
                    {(job.alerts_sent || 0).toLocaleString()}
                  </td>
                  <td style={{ padding: '0.5rem' }}>{formatTime(job.started_at)}</td>
                  <td style={{ padding: '0.5rem' }}>{formatTime(job.completed_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

export default AlertGenerator;
