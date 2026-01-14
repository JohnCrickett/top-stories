import React, { useState, useEffect } from 'react';
import './Stories.css';
import StoryItem from './StoryItem';

function Stories({ api, refreshTrigger }) {
  const [stories, setStories] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const fetchStories = async () => {
    if (!api) return;

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(api.url);
      if (!response.ok) throw new Error('Failed to fetch stories');
      const data = await response.json();
      setStories(data || []);
    } catch (err) {
      setError(`Failed to load stories from ${api.name}`);
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  // Fetch stories when api changes or refresh is triggered
  useEffect(() => {
    fetchStories();
  }, [api, refreshTrigger]);

  if (!api) {
    return <div className="stories">No API selected</div>;
  }

  if (loading && stories.length === 0) {
    return <div className="stories loading-state">Loading stories...</div>;
  }

  if (error) {
    return <div className="stories error-state">{error}</div>;
  }

  return (
    <div className="stories">
      {stories.length === 0 ? (
        <div className="no-stories">No stories available</div>
      ) : (
        <ol className="story-list">
          {stories.map((story, index) => (
            <StoryItem key={story.id} story={story} rank={index + 1} />
          ))}
        </ol>
      )}
    </div>
  );
}

export default Stories;
