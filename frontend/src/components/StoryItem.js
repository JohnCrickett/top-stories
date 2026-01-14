import React from 'react';
import './StoryItem.css';

function StoryItem({ story, rank }) {
  const timeAgo = getTimeAgo(story.time);
  const domain = extractDomain(story.url);

  return (
    <li className="story-item">
      <div className="story-row">
        <span className="rank">{rank}.</span>
        <div className="story-content">
          {story.url ? (
            <a href={story.url} target="_blank" rel="noopener noreferrer" className="story-title">
              {story.title}
            </a>
          ) : (
            <span className="story-title">{story.title}</span>
          )}
          {story.url && <span className="domain">({domain})</span>}
        </div>
      </div>
      <div className="story-meta">
        <span className="score">{story.score} points</span>
        {story.by && <span className="by">by {story.by}</span>}
        <span className="time">{timeAgo}</span>
      </div>
    </li>
  );
}

function getTimeAgo(timestamp) {
  const now = Math.floor(Date.now() / 1000);
  const diff = now - timestamp;

  if (diff < 60) return 'now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h`;
  if (diff < 604800) return `${Math.floor(diff / 86400)}d`;
  return `${Math.floor(diff / 604800)}w`;
}

function extractDomain(url) {
  try {
    const domain = new URL(url).hostname;
    return domain.replace('www.', '');
  } catch {
    return 'unknown';
  }
}

export default StoryItem;
