import { useState, useEffect } from 'react';
import { alertGeneratorAPI } from '../services/api';

function AlertGenerator() {
  const [activeJob, setActiveJob] = useState(null);
  const [jobHistory, setJobHistory] = useState([]);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  
  // Configuration state
  const [config, setConfig] = useState({
    rps: 10,
    duration: '60s',
    burst: null,
    seed: null,
    severity_dist: 'HIGH:30,MEDIUM:30,LOW:25,CRITICAL:15',
    source_dist: 'api:25,db:20,cache:15,monitor:15,queue:10,worker:5,frontend:5,backend:5',
    name_dist: 'timeout:15,error:15,crash:10,slow:10,memory:10,cpu:10,disk:10,network:10,auth:5,validation:5',
    kafka_brokers: 'localhost:9092',
    topic: 'alerts.new',
    mock: false,
    test: false,
    single_test: false,
  });

  // Load job history on mount
  useEffect(() => {
    loadJobHistory();
  }, []);

  // Poll active job status
  useEffect(() => {
    if (!activeJob) return;

    const interval = setInterval(async () => {
      try {
        const status = await alertGeneratorAPI.getStatus(activeJob.id);
        setActiveJob(status);
        
        if (status.status === 'completed' || status.status === 'failed' || status.status === 'cancelled') {
          clearInterval(interval);
          loadJobHistory();
          setActiveJob(null);
        }
      } catch (err) {
        console.error('Failed to fetch job status:', err);
      }
    }, 1000); // Poll every second

    return () => clearInterval(interval);
  }, [activeJob]);

  const loadJobHistory = async () => {
    try {
      const jobs = await alertGeneratorAPI.list();
      setJobHistory(jobs.slice(0, 10)); // Show last 10 jobs
    } catch (err) {
      console.error('Failed to load job history:', err);
      // Don't show error for initial load, but log it
      if (jobHistory.length === 0) {
        // Only set error if we haven't loaded any jobs yet
        setError('Unable to connect to alert-producer API. Make sure the API server is running on port 8082.');
      }
    }
  };

  const handlePreset = async (preset) => {
    setError(null);
    setLoading(true);
    
    try {
      let presetConfig = {};
      
      switch (preset) {
        case 'single-test':
          presetConfig = { single_test: true };
          break;
        case 'test-mode':
          presetConfig = { test: true, rps: 5, duration: '30s' };
          break;
        case 'burst-100':
          presetConfig = { burst: 100 };
          break;
        case 'burst-1000':
          presetConfig = { burst: 1000 };
          break;
        case 'load-test':
          presetConfig = { rps: 50, duration: '5m' };
          break;
        default:
          return;
      }
      
      const response = await alertGeneratorAPI.generate(presetConfig);
      const status = await alertGeneratorAPI.getStatus(response.job_id);
      setActiveJob(status);
      loadJobHistory();
    } catch (err) {
      console.error('Error starting alert generation:', err);
      let errorMessage = 'Failed to start alert generation';
      if (err.message) {
        if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError') || err.message.includes('ERR_CONNECTION_REFUSED')) {
          errorMessage = 'Cannot connect to alert-producer API. Make sure the API server is running:\n\ncd services/alert-producer && make run-api';
        } else {
          errorMessage = err.message;
        }
      }
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleGenerate = async (e) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    
    try {
      // Build config object, only including non-empty values
      const requestConfig = {};
      if (config.rps) requestConfig.rps = parseFloat(config.rps);
      if (config.duration) requestConfig.duration = config.duration;
      if (config.burst) requestConfig.burst = parseInt(config.burst);
      if (config.seed) requestConfig.seed = parseInt(config.seed);
      if (config.severity_dist) requestConfig.severity_dist = config.severity_dist;
      if (config.source_dist) requestConfig.source_dist = config.source_dist;
      if (config.name_dist) requestConfig.name_dist = config.name_dist;
      if (config.kafka_brokers) requestConfig.kafka_brokers = config.kafka_brokers;
      if (config.topic) requestConfig.topic = config.topic;
      if (config.mock) requestConfig.mock = config.mock;
      if (config.test) requestConfig.test = config.test;
      if (config.single_test) requestConfig.single_test = config.single_test;
      
      const response = await alertGeneratorAPI.generate(requestConfig);
      const status = await alertGeneratorAPI.getStatus(response.job_id);
      setActiveJob(status);
      loadJobHistory();
    } catch (err) {
      console.error('Error starting alert generation:', err);
      let errorMessage = 'Failed to start alert generation';
      if (err.message) {
        if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError') || err.message.includes('ERR_CONNECTION_REFUSED')) {
          errorMessage = 'Cannot connect to alert-producer API. Make sure the API server is running:\n\ncd services/alert-producer && make run-api';
        } else {
          errorMessage = err.message;
        }
      }
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    if (!activeJob) return;
    
    try {
      await alertGeneratorAPI.stop(activeJob.id);
      const status = await alertGeneratorAPI.getStatus(activeJob.id);
      setActiveJob(status);
      loadJobHistory();
    } catch (err) {
      console.error('Error stopping job:', err);
      let errorMessage = 'Failed to stop job';
      if (err.message) {
        if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError') || err.message.includes('ERR_CONNECTION_REFUSED')) {
          errorMessage = 'Cannot connect to alert-producer API. Make sure the API server is running.';
        } else {
          errorMessage = err.message;
        }
      }
      setError(errorMessage);
    }
  };

  const formatTime = (timeStr) => {
    if (!timeStr) return '-';
    return new Date(timeStr).toLocaleString();
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed': return '#28a745';
      case 'running': return '#007bff';
      case 'failed': return '#dc3545';
      case 'cancelled': return '#6c757d';
      default: return '#6c757d';
    }
  };

  return (
    <div className="alert-generator">
      <h2>Alert Generator</h2>
      
      {error && (
        <div className="error" style={{ color: '#dc3545', marginBottom: '1rem', padding: '0.5rem', background: '#f8d7da', borderRadius: '4px', whiteSpace: 'pre-line' }}>
          {error}
        </div>
      )}

      {/* Preset Buttons */}
      <div className="presets" style={{ marginBottom: '2rem' }}>
        <h3>Quick Start</h3>
        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          <button 
            onClick={() => handlePreset('single-test')} 
            disabled={loading || activeJob}
            style={{ padding: '0.5rem 1rem', cursor: loading || activeJob ? 'not-allowed' : 'pointer' }}
          >
            Single Test Alert
          </button>
          <button 
            onClick={() => handlePreset('test-mode')} 
            disabled={loading || activeJob}
            style={{ padding: '0.5rem 1rem', cursor: loading || activeJob ? 'not-allowed' : 'pointer' }}
          >
            Test Mode (5 RPS, 30s)
          </button>
          <button 
            onClick={() => handlePreset('burst-100')} 
            disabled={loading || activeJob}
            style={{ padding: '0.5rem 1rem', cursor: loading || activeJob ? 'not-allowed' : 'pointer' }}
          >
            Burst 100 Alerts
          </button>
          <button 
            onClick={() => handlePreset('burst-1000')} 
            disabled={loading || activeJob}
            style={{ padding: '0.5rem 1rem', cursor: loading || activeJob ? 'not-allowed' : 'pointer' }}
          >
            Burst 1000 Alerts
          </button>
          <button 
            onClick={() => handlePreset('load-test')} 
            disabled={loading || activeJob}
            style={{ padding: '0.5rem 1rem', cursor: loading || activeJob ? 'not-allowed' : 'pointer' }}
          >
            Load Test (50 RPS, 5m)
          </button>
        </div>
      </div>

      {/* Manual Configuration Form */}
      <div className="manual-config">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <h3>Manual Configuration</h3>
          <button 
            onClick={() => setShowAdvanced(!showAdvanced)}
            style={{ padding: '0.25rem 0.5rem' }}
          >
            {showAdvanced ? 'Hide' : 'Show'} Advanced
          </button>
        </div>

        <form onSubmit={handleGenerate}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem', marginBottom: '1rem' }}>
            <div>
              <label>RPS (Alerts per second):</label>
              <input
                type="number"
                step="0.1"
                value={config.rps}
                onChange={(e) => setConfig({ ...config, rps: e.target.value })}
                disabled={loading || activeJob}
              />
            </div>
            <div>
              <label>Duration (e.g., 60s, 5m):</label>
              <input
                type="text"
                value={config.duration}
                onChange={(e) => setConfig({ ...config, duration: e.target.value })}
                disabled={loading || activeJob}
                placeholder="60s"
              />
            </div>
            <div>
              <label>Burst Size (0 = continuous):</label>
              <input
                type="number"
                value={config.burst || ''}
                onChange={(e) => setConfig({ ...config, burst: e.target.value || null })}
                disabled={loading || activeJob}
                placeholder="0"
              />
            </div>
            <div>
              <label>Seed (0 = random):</label>
              <input
                type="number"
                value={config.seed || ''}
                onChange={(e) => setConfig({ ...config, seed: e.target.value || null })}
                disabled={loading || activeJob}
                placeholder="0"
              />
            </div>
          </div>

          {showAdvanced && (
            <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: '1rem', marginBottom: '1rem' }}>
              <div>
                <label>Severity Distribution:</label>
                <input
                  type="text"
                  value={config.severity_dist}
                  onChange={(e) => setConfig({ ...config, severity_dist: e.target.value })}
                  disabled={loading || activeJob}
                  placeholder="HIGH:30,MEDIUM:30,LOW:25,CRITICAL:15"
                />
              </div>
              <div>
                <label>Source Distribution:</label>
                <input
                  type="text"
                  value={config.source_dist}
                  onChange={(e) => setConfig({ ...config, source_dist: e.target.value })}
                  disabled={loading || activeJob}
                  placeholder="api:25,db:20,..."
                />
              </div>
              <div>
                <label>Name Distribution:</label>
                <input
                  type="text"
                  value={config.name_dist}
                  onChange={(e) => setConfig({ ...config, name_dist: e.target.value })}
                  disabled={loading || activeJob}
                  placeholder="timeout:15,error:15,..."
                />
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
                <div>
                  <label>Kafka Brokers:</label>
                  <input
                    type="text"
                    value={config.kafka_brokers}
                    onChange={(e) => setConfig({ ...config, kafka_brokers: e.target.value })}
                    disabled={loading || activeJob}
                  />
                </div>
                <div>
                  <label>Topic:</label>
                  <input
                    type="text"
                    value={config.topic}
                    onChange={(e) => setConfig({ ...config, topic: e.target.value })}
                    disabled={loading || activeJob}
                  />
                </div>
              </div>
              <div style={{ display: 'flex', gap: '1rem' }}>
                <label>
                  <input
                    type="checkbox"
                    checked={config.mock}
                    onChange={(e) => setConfig({ ...config, mock: e.target.checked })}
                    disabled={loading || activeJob}
                  />
                  Mock Mode (no Kafka)
                </label>
                <label>
                  <input
                    type="checkbox"
                    checked={config.test}
                    onChange={(e) => setConfig({ ...config, test: e.target.checked })}
                    disabled={loading || activeJob}
                  />
                  Test Mode
                </label>
                <label>
                  <input
                    type="checkbox"
                    checked={config.single_test}
                    onChange={(e) => setConfig({ ...config, single_test: e.target.checked })}
                    disabled={loading || activeJob}
                  />
                  Single Test
                </label>
              </div>
            </div>
          )}

          <button 
            type="submit" 
            disabled={loading || activeJob}
            style={{ 
              padding: '0.75rem 1.5rem', 
              fontSize: '1rem',
              cursor: loading || activeJob ? 'not-allowed' : 'pointer'
            }}
          >
            {loading ? 'Starting...' : 'Generate Alerts'}
          </button>
        </form>
      </div>

      {/* Active Job Status */}
      {activeJob && (
        <div className="active-job" style={{ marginTop: '2rem', padding: '1rem', border: '1px solid #ddd', borderRadius: '4px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
            <h3>Active Job: {activeJob.id}</h3>
            {activeJob.status === 'running' && (
              <button onClick={handleStop} style={{ padding: '0.5rem 1rem', background: '#dc3545', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer' }}>
                Stop
              </button>
            )}
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '1rem' }}>
            <div>
              <strong>Status:</strong> 
              <span style={{ color: getStatusColor(activeJob.status), marginLeft: '0.5rem' }}>
                {activeJob.status.toUpperCase()}
              </span>
            </div>
            <div>
              <strong>Alerts Sent:</strong> {activeJob.alerts_sent || 0}
            </div>
            <div>
              <strong>Started:</strong> {formatTime(activeJob.started_at)}
            </div>
            {activeJob.completed_at && (
              <div>
                <strong>Completed:</strong> {formatTime(activeJob.completed_at)}
              </div>
            )}
          </div>
          {activeJob.error && (
            <div style={{ marginTop: '0.5rem', color: '#dc3545' }}>
              <strong>Error:</strong> {activeJob.error}
            </div>
          )}
        </div>
      )}

      {/* Job History */}
      {jobHistory.length > 0 && (
        <div className="job-history" style={{ marginTop: '2rem' }}>
          <h3>Recent Jobs</h3>
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{ borderBottom: '2px solid #ddd' }}>
                <th style={{ textAlign: 'left', padding: '0.5rem' }}>Job ID</th>
                <th style={{ textAlign: 'left', padding: '0.5rem' }}>Status</th>
                <th style={{ textAlign: 'left', padding: '0.5rem' }}>Alerts Sent</th>
                <th style={{ textAlign: 'left', padding: '0.5rem' }}>Created</th>
                <th style={{ textAlign: 'left', padding: '0.5rem' }}>Completed</th>
              </tr>
            </thead>
            <tbody>
              {jobHistory.map((job) => (
                <tr key={job.id} style={{ borderBottom: '1px solid #eee' }}>
                  <td style={{ padding: '0.5rem', fontFamily: 'monospace', fontSize: '0.9em' }}>
                    {job.id.substring(0, 8)}...
                  </td>
                  <td style={{ padding: '0.5rem' }}>
                    <span style={{ color: getStatusColor(job.status) }}>
                      {job.status.toUpperCase()}
                    </span>
                  </td>
                  <td style={{ padding: '0.5rem' }}>{job.alerts_sent || 0}</td>
                  <td style={{ padding: '0.5rem' }}>{formatTime(job.created_at)}</td>
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
