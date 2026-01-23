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
    kafka_brokers: '',  // Leave empty to use server default
    topic: 'alerts.new',
    mock: false,
    test: false,
    single_test: false,
    // Single alert properties
    severity: 'LOW',
    source: 'test-source',
    name: 'test-name',
  });

  // Load job history and restore active job on mount
  useEffect(() => {
    loadJobHistory();
    
    // Restore active job from localStorage
    const savedJobId = localStorage.getItem('alertGenerator_activeJobId');
    if (savedJobId) {
      restoreActiveJob(savedJobId);
    }
  }, []);

  // Save active job ID to localStorage and poll status
  useEffect(() => {
    if (activeJob) {
      // Save to localStorage
      localStorage.setItem('alertGenerator_activeJobId', activeJob.id);
    } else {
      // Clear from localStorage when no active job
      localStorage.removeItem('alertGenerator_activeJobId');
    }

    if (!activeJob) return;

    const interval = setInterval(async () => {
      try {
        const status = await alertGeneratorAPI.getStatus(activeJob.id);
        setActiveJob(status);
        
        if (status.status === 'completed' || status.status === 'failed' || status.status === 'cancelled') {
          clearInterval(interval);
          loadJobHistory();
          setActiveJob(null);
          localStorage.removeItem('alertGenerator_activeJobId');
        }
      } catch (err) {
        console.error('Failed to fetch job status:', err);
        // If job not found, clear it
        if (err.message && err.message.includes('not found')) {
          clearInterval(interval);
          setActiveJob(null);
          localStorage.removeItem('alertGenerator_activeJobId');
        }
      }
    }, 500); // Poll every 500ms for better progress updates

    return () => clearInterval(interval);
  }, [activeJob]);

  const restoreActiveJob = async (jobId) => {
    try {
      const status = await alertGeneratorAPI.getStatus(jobId);
      // Only restore if job is still running
      if (status.status === 'running' || status.status === 'pending') {
        setActiveJob(status);
      } else {
        // Job completed, remove from localStorage
        localStorage.removeItem('alertGenerator_activeJobId');
      }
    } catch (err) {
      console.error('Failed to restore active job:', err);
      localStorage.removeItem('alertGenerator_activeJobId');
    }
  };

  // Helper function to extract user-friendly error message
  const getErrorMessage = (err) => {
    if (!err || !err.message) {
      return 'An unexpected error occurred';
    }
    const msg = err.message;
    if (msg.includes('Failed to fetch') || msg.includes('NetworkError') || msg.includes('ERR_CONNECTION_REFUSED')) {
      return 'Cannot connect to alert-producer API. Make sure the API server is running:\n\ncd services/alert-producer && make run-api';
    }
    return msg;
  };

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
      
      // Set preset-specific values
      switch (preset) {
        case 'single-test':
          // Use current config values for single alert
          presetConfig = { 
            single_test: true,
            severity: config.severity || 'LOW',
            source: config.source || 'test-source',
            name: config.name || 'test-name'
          };
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
      
      // Merge with advanced configuration settings
      // This allows presets to respect user's advanced settings
      if (config.severity_dist && config.severity_dist.trim() !== '' && !presetConfig.single_test) {
        presetConfig.severity_dist = config.severity_dist.trim();
      }
      if (config.source_dist && config.source_dist.trim() !== '' && !presetConfig.single_test) {
        presetConfig.source_dist = config.source_dist.trim();
      }
      if (config.name_dist && config.name_dist.trim() !== '' && !presetConfig.single_test) {
        presetConfig.name_dist = config.name_dist.trim();
      }
      
      // Always include Kafka settings if configured
      if (config.kafka_brokers && config.kafka_brokers.trim() !== '') {
        presetConfig.kafka_brokers = config.kafka_brokers.trim();
      }
      if (config.topic && config.topic.trim() !== '') {
        presetConfig.topic = config.topic.trim();
      }
      
      // Include seed if configured
      if (config.seed !== null && config.seed !== '' && !isNaN(parseInt(config.seed))) {
        presetConfig.seed = parseInt(config.seed);
      }
      
      // Include mock mode if enabled
      if (config.mock) {
        presetConfig.mock = true;
      }
      
      console.log('Sending preset config with advanced settings:', presetConfig);
      
      const response = await alertGeneratorAPI.generate(presetConfig);
      const status = await alertGeneratorAPI.getStatus(response.job_id);
      setActiveJob(status);
      loadJobHistory();
    } catch (err) {
      console.error('Error starting alert generation:', err);
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleGenerate = async (e) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    
    try {
      // Build config object, properly handling empty/null values
      const requestConfig = {};
      
      // RPS: only include if it's a valid number > 0
      if (config.rps && !isNaN(parseFloat(config.rps)) && parseFloat(config.rps) > 0) {
        requestConfig.rps = parseFloat(config.rps);
      }
      
      // Duration: only include if it's a non-empty string
      if (config.duration && config.duration.trim() !== '') {
        requestConfig.duration = config.duration.trim();
      }
      
      // Burst: only include if it's a valid integer > 0
      if (config.burst !== null && config.burst !== '' && !isNaN(parseInt(config.burst)) && parseInt(config.burst) > 0) {
        requestConfig.burst = parseInt(config.burst);
      }
      
      // Seed: only include if it's a valid integer
      if (config.seed !== null && config.seed !== '' && !isNaN(parseInt(config.seed))) {
        requestConfig.seed = parseInt(config.seed);
      }
      
      // Boolean flags: check first as they affect what other fields are needed
      requestConfig.mock = config.mock || false;
      requestConfig.test = config.test || false;
      requestConfig.single_test = config.single_test || false;
      
      // For single_test mode, include alert properties
      if (requestConfig.single_test) {
        if (config.severity && config.severity.trim() !== '') {
          requestConfig.severity = config.severity.trim();
        }
        if (config.source && config.source.trim() !== '') {
          requestConfig.source = config.source.trim();
        }
        if (config.name && config.name.trim() !== '') {
          requestConfig.name = config.name.trim();
        }
      }
      
      // For single_test mode, distributions are not needed (uses custom alert)
      // For test mode, distributions are optional (uses defaults if not provided)
      // For normal mode, distributions are required but have defaults
      if (!requestConfig.single_test) {
        // Distributions: only include if non-empty (will use defaults if not provided)
        if (config.severity_dist && config.severity_dist.trim() !== '') {
          requestConfig.severity_dist = config.severity_dist.trim();
        }
        if (config.source_dist && config.source_dist.trim() !== '') {
          requestConfig.source_dist = config.source_dist.trim();
        }
        if (config.name_dist && config.name_dist.trim() !== '') {
          requestConfig.name_dist = config.name_dist.trim();
        }
      }
      
      // Kafka settings: only include if non-empty
      if (config.kafka_brokers && config.kafka_brokers.trim() !== '') {
        requestConfig.kafka_brokers = config.kafka_brokers.trim();
      }
      if (config.topic && config.topic.trim() !== '') {
        requestConfig.topic = config.topic.trim();
      }
      
      console.log('Sending request config:', requestConfig);
      
      const response = await alertGeneratorAPI.generate(requestConfig);
      const status = await alertGeneratorAPI.getStatus(response.job_id);
      setActiveJob(status);
      loadJobHistory();
    } catch (err) {
      console.error('Error starting alert generation:', err);
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    if (!activeJob) return;
    
    try {
      await alertGeneratorAPI.stop(activeJob.id);
      // Poll once more to get updated status
      const status = await alertGeneratorAPI.getStatus(activeJob.id);
      setActiveJob(status);
      loadJobHistory();
      // Clear from localStorage if cancelled
      if (status.status === 'cancelled') {
        localStorage.removeItem('alertGenerator_activeJobId');
      }
    } catch (err) {
      console.error('Error stopping job:', err);
      setError(getErrorMessage(err));
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
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
          <h3 style={{ margin: 0 }}>Quick Start</h3>
          <span style={{ fontSize: '0.9em', color: '#6c757d' }}>
            Presets use your advanced configuration (distributions, Kafka settings, etc.)
          </span>
        </div>
        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          <button 
            type="button"
            onClick={() => {
              setConfig({ ...config, single_test: !config.single_test });
            }}
            disabled={loading || activeJob}
            style={{ 
              padding: '0.5rem 1rem', 
              cursor: loading || activeJob ? 'not-allowed' : 'pointer',
              background: config.single_test ? '#007bff' : '#f8f9fa',
              color: config.single_test ? 'white' : 'black',
              border: '1px solid #ddd'
            }}
          >
            {config.single_test ? '✓ Single Alert Mode' : 'Single Alert Mode'}
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
          {/* Single Alert Configuration - shown when single_test is checked */}
          {config.single_test && (
            <div style={{ 
              marginBottom: '1rem', 
              padding: '1rem', 
              background: '#e7f3ff', 
              borderRadius: '4px',
              border: '1px solid #b3d9ff'
            }}>
              <h4 style={{ marginTop: 0, marginBottom: '0.5rem' }}>Single Alert Configuration</h4>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '1rem' }}>
                <div>
                  <label>Severity:</label>
                  <select
                    value={config.severity}
                    onChange={(e) => setConfig({ ...config, severity: e.target.value })}
                    disabled={loading || activeJob}
                    style={{ width: '100%', padding: '0.5rem' }}
                  >
                    <option value="LOW">LOW</option>
                    <option value="MEDIUM">MEDIUM</option>
                    <option value="HIGH">HIGH</option>
                    <option value="CRITICAL">CRITICAL</option>
                  </select>
                </div>
                <div>
                  <label>Source:</label>
                  <input
                    type="text"
                    value={config.source}
                    onChange={(e) => setConfig({ ...config, source: e.target.value })}
                    disabled={loading || activeJob}
                    placeholder="e.g., api, db, cache"
                    style={{ width: '100%', padding: '0.5rem' }}
                  />
                </div>
                <div>
                  <label>Name:</label>
                  <input
                    type="text"
                    value={config.name}
                    onChange={(e) => setConfig({ ...config, name: e.target.value })}
                    disabled={loading || activeJob}
                    placeholder="e.g., timeout, error, crash"
                    style={{ width: '100%', padding: '0.5rem' }}
                  />
                </div>
              </div>
            </div>
          )}

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem', marginBottom: '1rem' }}>
            <div>
              <label>RPS (Alerts per second):</label>
              <input
                type="number"
                step="0.1"
                value={config.rps}
                onChange={(e) => setConfig({ ...config, rps: e.target.value })}
                disabled={loading || activeJob || config.single_test}
              />
            </div>
            <div>
              <label>Duration (e.g., 60s, 5m):</label>
              <input
                type="text"
                value={config.duration}
                onChange={(e) => setConfig({ ...config, duration: e.target.value })}
                disabled={loading || activeJob || config.single_test}
                placeholder="60s"
              />
            </div>
            <div>
              <label>Burst Size (0 = continuous):</label>
              <input
                type="number"
                value={config.burst || ''}
                onChange={(e) => setConfig({ ...config, burst: e.target.value || null })}
                disabled={loading || activeJob || config.single_test}
                placeholder="0"
              />
            </div>
            <div>
              <label>Seed (0 = random):</label>
              <input
                type="number"
                value={config.seed || ''}
                onChange={(e) => setConfig({ ...config, seed: e.target.value || null })}
                disabled={loading || activeJob || config.single_test}
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
        <div className="active-job" style={{ marginTop: '2rem', padding: '1rem', border: '2px solid #007bff', borderRadius: '4px', background: '#f8f9fa' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
            <h3 style={{ margin: 0 }}>Active Job: {activeJob.id.substring(0, 8)}...</h3>
            {activeJob.status === 'running' && (
              <button 
                onClick={handleStop} 
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
                Stop Job
              </button>
            )}
          </div>
          
          {/* Progress Bar for Running Jobs */}
          {activeJob.status === 'running' && activeJob.config && (
            <div style={{ marginBottom: '1rem' }}>
              {activeJob.config.burst && activeJob.config.burst > 0 ? (
                <>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.25rem' }}>
                    <span><strong>Progress:</strong> {activeJob.alerts_sent || 0} / {activeJob.config.burst}</span>
                    <span><strong>{Math.round(((activeJob.alerts_sent || 0) / activeJob.config.burst) * 100)}%</strong></span>
                  </div>
                  <div style={{ 
                    width: '100%', 
                    height: '20px', 
                    background: '#e9ecef', 
                    borderRadius: '10px', 
                    overflow: 'hidden' 
                  }}>
                    <div style={{
                      width: `${Math.min(((activeJob.alerts_sent || 0) / activeJob.config.burst) * 100, 100)}%`,
                      height: '100%',
                      background: '#28a745',
                      transition: 'width 0.3s ease'
                    }} />
                  </div>
                </>
              ) : (
                <div>
                  <strong>Alerts Sent:</strong> {activeJob.alerts_sent || 0}
                  {activeJob.config.rps && (
                    <span style={{ marginLeft: '1rem', color: '#6c757d' }}>
                      (Target: {activeJob.config.rps} RPS)
                    </span>
                  )}
                </div>
              )}
            </div>
          )}
          
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '1rem' }}>
            <div>
              <strong>Status:</strong> 
              <span style={{ color: getStatusColor(activeJob.status), marginLeft: '0.5rem', fontWeight: 'bold' }}>
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
            <div style={{ marginTop: '0.5rem', padding: '0.5rem', background: '#f8d7da', borderRadius: '4px', color: '#dc3545' }}>
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
