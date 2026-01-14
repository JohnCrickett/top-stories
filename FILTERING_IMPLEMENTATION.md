# Consumer-Side Filtering Implementation

## Overview

This implementation adds configurable consumer-side filtering to the story-api service, enabling each instance to consume and store only stories matching configured filters. Stories are filtered at ingest time before being stored in memory, making the system more scalable and efficient.

## Changes Made

### 1. Configuration (`story-api/config.yaml`)

Added a new `filter` section to the configuration:

```yaml
filter:
  story_types: []        # List of story types to consume (e.g., ["ask", "show"])
  keywords: []           # Keywords to match in titles (case-insensitive)
  minimum_score: 0       # Minimum score threshold
```

### 2. Core Implementation (`story-api/main.go`)

#### New Types

- **`FilterConfig`**: Configuration struct for filter settings
  - `StoryTypes`: List of story types to filter by
  - `Keywords`: List of keywords to search for in titles
  - `MinimumScore`: Minimum score threshold

- **`StoryFilter`**: Runtime filter implementation
  - `storyTypes`: Map for O(1) lookups of allowed types
  - `keywords`: Keywords list
  - `minimumScore`: Score threshold
  - `enabled`: Flag indicating if any filters are active

#### Key Methods

- **`NewStoryFilter(cfg FilterConfig)`**: Factory function that creates a filter from config
  - Converts story_types list to a map for efficient lookups
  - Determines if filtering is enabled (any constraint specified)

- **`Matches(story *Story) bool`**: Core filtering logic
  - Returns `true` if story passes all configured filters
  - Returns `true` if no filters are enabled (passthrough mode)
  - Implements AND logic for multiple constraints:
    - Story type must be in allowed list (if configured)
    - Score must be >= minimum_score (if configured)
    - Title must contain at least one keyword (if configured)

#### Integration Points

1. **Server Initialization**: `NewServer()` creates a filter from config
2. **Message Consumption**: `consumeMessages()` applies filter before storing
   - Non-matching stories are discarded at ingest time (reducing memory usage)
   - Kafka offset is still committed to advance the consumer

3. **Startup Logging**: `start()` logs active filter configuration for visibility

### 3. Example Configurations

Four example configuration files are provided in `story-api/`:

- **`config.ask-hn.yaml`**: Consumes only "ask" stories
  ```yaml
  filter:
    story_types: ["ask"]
    keywords: []
    minimum_score: 0
  ```

- **`config.rust.yaml`**: Consumes Rust-related stories
  ```yaml
  filter:
    story_types: []
    keywords: ["rust"]
    minimum_score: 0
  ```

- **`config.top.yaml`**: Consumes high-scoring stories
  ```yaml
  filter:
    story_types: []
    keywords: []
    minimum_score: 100
  ```

- **`config.show-hn.yaml`**: Consumes Show HN stories with score >= 10
  ```yaml
  filter:
    story_types: ["story"]
    keywords: ["show"]
    minimum_score: 10
  ```

## Design Details

### Filtering Logic

The `Matches()` method implements **AND semantics** across filters:
1. **Type Filter**: If configured, story type must match
2. **Keyword Filter**: If configured, story title must contain at least one keyword (case-insensitive)
3. **Score Filter**: If configured, story score must meet minimum threshold

All constraints must be satisfied for a story to pass (AND logic between filters).

### Performance Characteristics

- **Type lookups**: O(1) using map
- **Keyword matching**: O(k*t) where k=keywords, t=title length (substring search)
- **Score comparison**: O(1)
- **Overall**: O(1) type + O(k*t) keywords per message

Stories are filtered at ingest time, reducing:
- Memory footprint (non-matching stories never stored)
- API query load (queries operate on smaller dataset)
- Network traffic when returning stories

### No Filters (Passthrough Mode)

When no filters are configured:
- `filter.enabled` is `false`
- `Matches()` immediately returns `true`
- All stories are consumed (backward compatible with existing deployment)

### Kafka Offset Management

The filter is applied **before** storage but **after** consuming from Kafka:
- Offsets are committed regardless of whether story was stored
- Ensures no messages are re-consumed on restart
- Guarantees forward progress through the Kafka topic

## Running Multiple Instances

Each instance can be deployed with a different configuration:

```bash
# Terminal 1: All stories
story-api --config=config.yaml &

# Terminal 2: Ask HN stories
story-api --config=config.ask-hn.yaml &

# Terminal 3: Rust stories  
story-api --config=config.rust.yaml &

# Terminal 4: High-scoring stories
story-api --config=config.top.yaml &
```

Key benefits:
- Each instance only stores relevant stories (reduced memory/storage)
- API endpoints return specialized subsets
- Parallel processing across filtered topic subsets
- Independent scaling per instance based on story volume

## Testing

To verify the filtering works:

1. **Build**: `go build -o story-api`
2. **Test with no filters**: Stories pass through unchanged
3. **Test with story_types filter**: Only configured types are stored
4. **Test with keywords filter**: Only stories with matching keywords are stored
5. **Test with minimum_score filter**: Only stories above threshold are stored
6. **Check logs**: `[FILTERED]` messages show discarded stories, `[STORED]` shows accepted stories

## Backward Compatibility

The implementation is fully backward compatible:
- Default `config.yaml` has empty filter configuration
- Existing deployments work unchanged (all stories consumed)
- Filter is optional and defaults to disabled (passthrough)
- No changes to API endpoints or response format

## Future Enhancements

Possible extensions to the filtering system:
- Regular expression support for keywords
- Logical operators (AND/OR/NOT) for keywords
- Story author/by filtering
- URL domain filtering
- Dynamic filter updates via API (without restart)
- Filter statistics/metrics endpoint
