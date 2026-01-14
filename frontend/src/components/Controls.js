import React from 'react';
import './Controls.css';

function Controls({
  refreshInterval,
  setRefreshInterval,
  onRefresh,
  loading,
  lastRefreshTime,
}) {
  const formatRefreshTime = () => {
    if (!lastRefreshTime) return 'never';
    const now = new Date();
    const diff = Math.floor((now - lastRefreshTime) / 1000);
    if (diff < 60) return `${diff}s ago`;
    return `${Math.floor(diff / 60)}m ago`;
  };

  return (
    <div className="controls">
      <div className="control-group">
        <label htmlFor="interval">Refresh:</label>
        <input
          id="interval"
          type="number"
          min="1"
          max="300"
          value={refreshInterval}
          onChange={(e) => setRefreshInterval(parseInt(e.target.value) || 60)}
          className="interval-input"
        />
        <span className="interval-label">min</span>
      </div>
      <div className="control-group">
        <button
          onClick={onRefresh}
          disabled={loading}
          className="refresh-btn"
        >
          {loading ? 'Loading...' : 'Refresh'}
        </button>
      </div>
      <div className="last-refresh">
        Last updated: {formatRefreshTime()}
      </div>
    </div>
  );
}

export default Controls;
