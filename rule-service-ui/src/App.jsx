import { useState } from 'react';
import Clients from './components/Clients';
import Rules from './components/Rules';
import Endpoints from './components/Endpoints';
import Notifications from './components/Notifications';
import AlertGenerator from './components/AlertGenerator';
import Metrics from './components/Metrics';
import Services from './components/Services';
import './index.css';

function App() {
  const [activeTab, setActiveTab] = useState('dashboard');

  const navGroups = [
    {
      label: 'Monitoring',
      tabs: [
        { id: 'dashboard', label: 'Dashboard' },
        { id: 'metrics', label: 'Metrics' },
      ]
    },
    {
      label: 'Configuration',
      tabs: [
        { id: 'clients', label: 'Clients' },
        { id: 'rules', label: 'Rules' },
        { id: 'endpoints', label: 'Endpoints' },
        { id: 'notifications', label: 'Notifications' },
      ]
    },
    {
      label: 'Testing',
      tabs: [
        { id: 'alerts', label: 'Load Test' },
      ]
    }
  ];

  return (
    <div className="container">
      <div className="header">
        <h1>Alerting Platform</h1>
        <div className="nav">
          {navGroups.map((group, groupIdx) => (
            <div key={group.label} className="nav-group">
              <span className="nav-group-label">{group.label}</span>
              <div className="nav-group-buttons">
                {group.tabs.map(tab => (
                  <button
                    key={tab.id}
                    className={activeTab === tab.id ? 'active' : ''}
                    onClick={() => setActiveTab(tab.id)}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>
              {groupIdx < navGroups.length - 1 && <div className="nav-divider" />}
            </div>
          ))}
        </div>
      </div>

      {activeTab === 'dashboard' && <Metrics />}
      {activeTab === 'metrics' && <Services />}
      {activeTab === 'clients' && <Clients />}
      {activeTab === 'rules' && <Rules />}
      {activeTab === 'endpoints' && <Endpoints />}
      {activeTab === 'notifications' && <Notifications />}
      {activeTab === 'alerts' && <AlertGenerator />}
    </div>
  );
}

export default App;
