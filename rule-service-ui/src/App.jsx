import { useState } from 'react';
import Clients from './components/Clients';
import Rules from './components/Rules';
import Endpoints from './components/Endpoints';
import Notifications from './components/Notifications';
import './index.css';

function App() {
  const [activeTab, setActiveTab] = useState('clients');

  return (
    <div className="container">
      <div className="header">
        <h1>Rule Service Management</h1>
        <div className="nav">
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
        </div>
      </div>

      {activeTab === 'clients' && <Clients />}
      {activeTab === 'rules' && <Rules />}
      {activeTab === 'endpoints' && <Endpoints />}
      {activeTab === 'notifications' && <Notifications />}
    </div>
  );
}

export default App;
