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
  const [activeTab, setActiveTab] = useState('metrics');

  return (
    <div className="container">
      <div className="header">
        <h1>Alerting Platform</h1>
        <div className="nav">
          <button
            className={activeTab === 'metrics' ? 'active' : ''}
            onClick={() => setActiveTab('metrics')}
          >
            Dashboard
          </button>
          <button
            className={activeTab === 'services' ? 'active' : ''}
            onClick={() => setActiveTab('services')}
          >
            Services
          </button>
          <button
            className={activeTab === 'clients' ? 'active' : ''}
            onClick={() => setActiveTab('clients')}
          >
            Clients
          </button>
          <button
            className={activeTab === 'rules' ? 'active' : ''}
            onClick={() => setActiveTab('rules')}
          >
            Rules
          </button>
          <button
            className={activeTab === 'endpoints' ? 'active' : ''}
            onClick={() => setActiveTab('endpoints')}
          >
            Endpoints
          </button>
          <button
            className={activeTab === 'notifications' ? 'active' : ''}
            onClick={() => setActiveTab('notifications')}
          >
            Notifications
          </button>
          <button
            className={activeTab === 'alerts' ? 'active' : ''}
            onClick={() => setActiveTab('alerts')}
          >
            Alert Generator
          </button>
        </div>
      </div>

      {activeTab === 'metrics' && <Metrics />}
      {activeTab === 'services' && <Services />}
      {activeTab === 'clients' && <Clients />}
      {activeTab === 'rules' && <Rules />}
      {activeTab === 'endpoints' && <Endpoints />}
      {activeTab === 'notifications' && <Notifications />}
      {activeTab === 'alerts' && <AlertGenerator />}
    </div>
  );
}

export default App;
