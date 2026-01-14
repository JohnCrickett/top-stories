import React from 'react';
import './Tabs.css';

function Tabs({ apis, activeTab, setActiveTab }) {
  return (
    <div className="tabs">
      {apis.map((api, index) => (
        <button
          key={index}
          className={`tab ${activeTab === index ? 'active' : ''}`}
          onClick={() => setActiveTab(index)}
        >
          {api.name}
        </button>
      ))}
    </div>
  );
}

export default Tabs;
