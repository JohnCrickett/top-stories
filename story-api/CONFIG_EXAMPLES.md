# Consumer-Side Filtering Configuration Examples

## Basic Configuration Structure

All filter options go in the `filter` section of `config.yaml`:

```yaml
kafka:
  broker: kafka-broker:19196
  topic: hn-stories
  consumer_group: story-api-default
  ca_cert_path: ../certs-and-keys/ca.pem
  client_cert_path: ../certs-and-keys/service.cert
  client_key_path: ../certs-and-keys/service.key

api:
  port: 8080

# Consumer-side filtering
filter:
  story_types: []      # Story types to filter by
  keywords: []         # Keywords to match in titles
  minimum_score: 0     # Minimum score threshold
```

## Example Configurations

### 1. No Filtering (Default - Consume All Stories)

```yaml
filter:
  story_types: []
  keywords: []
  minimum_score: 0
```

**Result**: All stories from Hacker News are consumed and stored.

---

### 2. Ask HN Stories Only

```yaml
filter:
  story_types: ["ask"]
  keywords: []
  minimum_score: 0
```

**Result**: Only stories where `type == "ask"` are stored.

**Configuration**: `config.ask-hn.yaml`

**Use case**: Dedicated instance for "Ask HN" stories.

---

### 3. Show HN Stories Only

```yaml
filter:
  story_types: ["story"]
  keywords: ["show"]
  minimum_score: 0
```

**Result**: Only stories of type "story" with "show" in the title.

**Configuration**: `config.show-hn.yaml`

**Use case**: Dedicated instance for "Show HN" stories.

---

### 4. Keyword-Based Filtering (Rust Stories)

```yaml
filter:
  story_types: []
  keywords: ["rust"]
  minimum_score: 0
```

**Result**: Any story with "rust" (case-insensitive) in the title.

**Configuration**: `config.rust.yaml`

**Example stories**:
- "Rust 1.75 Released"
- "Building Web Apps in Rust"
- "Why Rust for Systems Programming"
- (Won't match: "The Rustic Café" - would need to be more specific)

**Use case**: Topic-specific instances for programming languages or technologies.

---

### 5. Minimum Score Filtering (Popular Stories)

```yaml
filter:
  story_types: []
  keywords: []
  minimum_score: 100
```

**Result**: Only stories with `score >= 100` are stored.

**Configuration**: `config.top.yaml`

**Use case**: High-quality, popular stories only.

---

### 6. Multiple Keywords (Any Match)

```yaml
filter:
  story_types: []
  keywords: ["python", "javascript", "go", "rust"]
  minimum_score: 0
```

**Result**: Stories about programming languages (ANY keyword match).

**Example stories that match**:
- "Python 3.12 Features"
- "JavaScript Performance Tips"
- "Building CLI Tools in Go"
- "Rust WebAssembly Tutorial"

**Use case**: Aggregate stories for multiple topics.

---

### 7. Combined Filtering (Type + Keywords + Score)

```yaml
filter:
  story_types: ["story"]
  keywords: ["ai", "machine learning", "llm"]
  minimum_score: 50
```

**Result**: Stories of type "story" with AI-related keywords AND score >= 50.

**Logic**: `(type == "story") AND (title contains "ai" OR "machine learning" OR "llm") AND (score >= 50)`

**Use case**: Curated feed for specific topic with quality threshold.

---

### 8. Poll Stories Only

```yaml
filter:
  story_types: ["poll"]
  keywords: []
  minimum_score: 0
```

**Result**: Only stories where `type == "poll"`.

**Use case**: Instance dedicated to Hacker News polls.

---

### 9. Job Posts Only

```yaml
filter:
  story_types: ["job"]
  keywords: []
  minimum_score: 0
```

**Result**: Only job postings.

**Use case**: Dedicated job board for Hacker News.

---

### 10. High-Scoring Non-Ask Stories

```yaml
filter:
  story_types: ["story", "show", "poll"]
  keywords: []
  minimum_score: 50
```

**Result**: Only high-quality stories, excluding Ask HN and Job posts.

**Use case**: "Best of" feed excluding certain categories.

---

## Hacker News Story Types

The Hacker News API provides the following story types:

- `story` - Regular stories/articles
- `ask` - "Ask HN" questions
- `show` - "Show HN" project/creation posts
- `poll` - Polls
- `job` - Job postings
- `pollopt` - Poll options (usually filtered out)

## Filtering Logic (AND Semantics)

When multiple filters are configured, ALL must be satisfied:

```
story_types filter AND keywords filter AND minimum_score filter
```

### Type Filter Logic
- If `story_types` is empty: ✓ (all types pass)
- If `story_types` is non-empty: ✓ (only if story type is in list)

### Keywords Filter Logic
- If `keywords` is empty: ✓ (all stories pass)
- If `keywords` is non-empty: ✓ (only if title contains ANY keyword, case-insensitive)

### Score Filter Logic
- If `minimum_score` is 0: ✓ (all scores pass)
- If `minimum_score` > 0: ✓ (only if score >= minimum_score)

## Performance Considerations

1. **Type matching**: O(1) - converted to map for fast lookup
2. **Keyword matching**: O(k*t) where k=keywords, t=title length
3. **Score comparison**: O(1)

**Total per message**: ~O(1) + O(k*t)

Stories are filtered at ingest time, so:
- Non-matching stories never stored in memory
- Reduced API query load
- Lower network bandwidth when returning results

## Real-World Deployment Example

```bash
# Instance 1: All stories (default)
./story-api &
# Serves: http://localhost:8080/stories

# Instance 2: Ask HN
./story-api -config config.ask-hn.yaml &
# Serves: http://localhost:8081/stories
# Content: Only "Ask HN" stories

# Instance 3: Popular Rust content
./story-api -config config.rust.yaml &
# Serves: http://localhost:8082/stories
# Content: Only Rust-related stories

# Instance 4: High-quality content
./story-api -config config.top.yaml &
# Serves: http://localhost:8083/stories
# Content: Only stories with score >= 100
```

Or run all at once:
```bash
./run-instances.sh
```

Each instance:
- Connects to the same Kafka broker/topic
- Maintains its own consumer group
- Filters at ingest time
- Serves only filtered results from its API

## Monitoring

Check logs for filter activity:

```
[CONFIG] Consumer-side filtering enabled:
  Story types: [ask]
  Keywords: []
  Minimum score: 0

[STORED] Story ID 35682445: Ask HN: How to learn Rust? (Score: 42)
[FILTERED] Story ID 35682446: New JavaScript Framework (Type: story, Score: 15)
```

- `[STORED]`: Story matched filters and is now available via API
- `[FILTERED]`: Story didn't match filters and was discarded
