import React, { useState, useEffect } from 'react';
import './App.css';
import Tabs from './components/Tabs';
import Stories from './components/Stories';
import Controls from './components/Controls';

function App() {
  const [apis, setApis] = useState([]);
  const [activeTab, setActiveTab] = useState(0);
  const [refreshInterval, setRefreshInterval] = useState(60);
  const [lastRefreshTime, setLastRefreshTime] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Fetch API config from backend
  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const response = await fetch('/api/config');
        if (!response.ok) throw new Error('Failed to fetch config');
        const data = await response.json();
        setApis(data.apis || []);
        if (data.refreshInterval) {
          setRefreshInterval(data.refreshInterval);
        }
        setError(null);
      } catch (err) {
        setError('Failed to load API configuration');
        console.error(err);
      }
    };

    fetchConfig();
  }, []);

  // Auto-refresh stories at configured interval
  useEffect(() => {
    const timer = setInterval(() => {
      handleRefresh();
    }, refreshInterval * 1000);

    return () => clearInterval(timer);
  }, [refreshInterval]);

  const handleRefresh = async () => {
    if (apis.length === 0) return;

    setLoading(true);
    try {
      // Trigger refresh for all story components
      // This is handled by the Stories component directly
      setLastRefreshTime(new Date());
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="app">
      <header className="header">
        <h1>Top Stories</h1>
        <Controls
          refreshInterval={refreshInterval}
          setRefreshInterval={setRefreshInterval}
          onRefresh={handleRefresh}
          loading={loading}
          lastRefreshTime={lastRefreshTime}
        />
      </header>

      {error && <div className="error-banner">{error}</div>}

      {apis.length > 0 ? (
        <div className="container">
          <Tabs
            apis={apis}
            activeTab={activeTab}
            setActiveTab={setActiveTab}
          />
          <div className="content">
            <Stories
              api={apis[activeTab]}
              refreshTrigger={lastRefreshTime}
            />
          </div>
        </div>
      ) : (
        <div className="loading">
          <p>Loading API configuration...</p>
        </div>
      )}
    </div>
  );
}

export default App;
