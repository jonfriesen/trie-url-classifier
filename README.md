# Trie-Based URL Pattern Classifier

[![Go Reference](https://pkg.go.dev/badge/github.com/ZoriHQ/trie-url-classifier.svg)](https://pkg.go.dev/github.com/ZoriHQ/trie-url-classifier)
[![Go Report Card](https://goreportcard.com/badge/github.com/ZoriHQ/trie-url-classifier)](https://goreportcard.com/report/github.com/ZoriHQ/trie-url-classifier)

A Go library that automatically learns URL patterns from batches of URLs and normalizes them by detecting dynamic segments. Zero external dependencies.

## Why This Library?

We built this library to normalize URLs at [Zori](https://zorihq.com) when processing customer analytics events. High-cardinality URLs with dynamic IDs, UUIDs, and dates create storage and query performance issues that this library solves by:

- **Automatic pattern detection** - No manual route definitions needed
- **Batch-based learning** - Discovers patterns from actual URLs in each batch
- **Cardinality reduction** - Converts `/users/12345/profile` to `/users/{id}/profile`
- **Type-aware** - Distinguishes UUIDs, IDs, dates, hashes, and other parameter types

Works for analytics, metrics storage, log normalization, and any system that needs to aggregate URL-based data.

## Key Features

- **Batch-Based Processing**: Create a new classifier instance per batch to avoid memory growth
- **Live Learning Mode**: Automatically learn patterns during classification until threshold is reached
- **Smart Pattern Detection**: Detects high-cardinality segments automatically
- **Parameter Type Classification**: Identifies UUIDs, IDs, dates, timestamps, hashes, tokens, and slugs
- **No Upfront Knowledge**: Learns patterns from actual URLs in each batch
- **Thread-Safe**: Safe for concurrent use with RWMutex protection
- **Configurable**: Adjust cardinality thresholds and minimum sample requirements

## Installation

```bash
go get github.com/ZoriHQ/trie-url-classifier
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    classifier "github.com/ZoriHQ/trie-url-classifier"
)

func main() {
    // Process a batch of URLs
    batch := []string{
        "/projects/d381b052-99eb-40f2-9ede-9bce790faae1/analytics",
        "/projects/a1b2c3d4-e5f6-7890-abcd-ef1234567890/analytics",
        "/projects/12345678-1234-1234-1234-123456789012/analytics",
        "/users/123456/profile",
        "/users/789012/profile",
        "/users/345678/profile",
    }

    // Create a NEW classifier for this batch
    c := classifier.NewClassifier()

    // Learn patterns from the batch
    c.Learn(batch)

    // Normalize URLs in the batch
    for _, url := range batch {
        pattern, err := c.Classify(url)
        if err != nil {
            log.Printf("Still learning: %v", err)
            continue
        }
        fmt.Printf("%s -> %s\n", url, pattern)
    }

    // After processing, discard the classifier
    // For the next batch, create a NEW instance
}
```

## Usage Pattern

### Batch Processing

Process batches of URLs independently:

```go
func processBatch(urlBatch []string) {
    c := classifier.NewClassifier()
    c.Learn(urlBatch)

    for _, url := range urlBatch {
        normalizedURL, err := c.Classify(url)
        if err != nil {
            // Handle error (only occurs in live learning mode)
            continue
        }
        // Use normalizedURL for storage, aggregation, etc.
    }
}
```

**Important**: Create a new classifier for each batch to avoid memory growth.

### Live Learning Mode

For streaming scenarios where you want to learn and classify in one step:

```go
// Create classifier with minimum learning count
c := classifier.NewClassifier(
    classifier.WithMinLearningCount(100), // Learn 100 URLs before classifying
)

for url := range urlStream {
    pattern, err := c.Classify(url)
    if err != nil {
        // Still learning - err contains current count
        insuffErr := err.(*classifier.InsufficientDataError)
        fmt.Printf("Learning: %d URLs processed\n", insuffErr.Count)
        continue
    }
    // Threshold reached - now classifying
    fmt.Printf("%s -> %s\n", url, pattern)
}
```

When `MinLearningCount` is set:
- `Classify()` automatically inserts URLs until the threshold is reached
- Returns `InsufficientDataError` with the current count during learning
- After threshold, returns normalized patterns normally

### Continued Learning Mode

`Classify()` always learns the URL and then classifies it. Memory is bounded by the pruning options below.

```go
c := classifier.NewClassifier(
    classifier.WithMinLearningCount(100),      // Learn 100 URLs before classifying
    classifier.WithMaxValuesPerNode(100),      // Bound memory per node
    classifier.WithPruneHighCardinality(true), // Collapse high-cardinality nodes
)

for url := range urlStream {
    pattern, err := c.Classify(url)
    if err != nil {
        // Below min threshold - still learning only
        continue
    }
    fmt.Printf("%s -> %s\n", url, pattern)
}
```

| URL Count | Behavior |
|-----------|----------|
| `< MinLearningCount` | Learn only, return `InsufficientDataError` |
| `>= MinLearningCount` | Learn AND classify |

## Configuration

Customize the classifier behavior:

```go
c := classifier.NewClassifier(
    classifier.WithCardinalityThreshold(0.75),  // 75% unique values = dynamic (default)
    classifier.WithMinSamples(2),               // Minimum samples before detection (default: 2)
    classifier.WithMinLearningCount(0),         // URLs to learn before classifying (default: 0)
    classifier.WithMaxValuesPerNode(100),       // Cap values per node for memory (default: 0 = unlimited)
    classifier.WithPruneHighCardinality(true),  // Collapse high-cardinality nodes (default: false)
)
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithCardinalityThreshold(float64)` | 0.75 | Ratio of unique values to total count. Higher = stricter detection |
| `WithMinSamples(int)` | 2 | Minimum samples needed at a position before considering it for parametrization |
| `WithMinLearningCount(int)` | 0 | Minimum URLs to learn before `Classify()` returns patterns |
| `WithMaxValuesPerNode(int)` | 0 | Maximum unique values to track per node. Bounds memory growth. 0 = unlimited |
| `WithPruneHighCardinality(bool)` | false | Collapse high-cardinality children into wildcard nodes to bound memory |

## Parameter Type Detection

The classifier automatically detects and labels these parameter types:

| Type | Pattern | Example |
|------|---------|---------|
| `{uuid}` | UUID v4 format | `d381b052-99eb-40f2-9ede-9bce790faae1` |
| `{id}` | Numeric ID (6+ digits) or prefixed IDs | `123456`, `cus_abc123` |
| `{hash}` | 24+ hex characters | `507f1f77bcf86cd799439011` |
| `{date}` | ISO date (YYYY-MM-DD) | `2024-01-15` |
| `{timestamp}` | Unix timestamp (10+ digits) | `1705334400` |
| `{token}` | JWT tokens | `eyJhbGci...` |
| `{slug}` | Hyphenated words with numbers | `my-post-12345` |
| `{param}` | Generic parameter (fallback) | Any other dynamic value |

## How It Works

1. **Build Trie**: URLs are split by `/` and inserted into a trie structure
2. **Track Cardinality**: Each node tracks unique values and total count
3. **Detect Patterns**: High-cardinality positions (many unique values) = dynamic segments
4. **Classify Types**: Dynamic values are classified using regex patterns
5. **Normalize**: New URLs are matched against learned patterns

### Example

Given training URLs:
```
/projects/uuid-1/analytics
/projects/uuid-2/analytics
/projects/uuid-3/analytics
```

The trie detects:
- "projects" appears 3 times with same value → static
- Second segment has 3 unique values in 3 traversals → dynamic (100% cardinality)
- "analytics" appears 3 times with same value → static

Result: `/projects/{uuid}/analytics`

## API Reference

### `NewClassifier(opts ...Option) *Classifier`

Creates a new URL pattern classifier with optional configuration.

### `(*Classifier) Learn(urls []string)`

Learns patterns from a batch of URLs. Can be called multiple times. Thread-safe.

### `(*Classifier) Classify(url string) (string, error)`

Normalizes a URL based on learned patterns. Thread-safe.

**Returns:**
- `(pattern, nil)` - Successfully classified URL
- `("", *InsufficientDataError)` - Still in learning phase (when `MinLearningCount` > 0)

### `InsufficientDataError`

Error returned when `Classify()` is called before `MinLearningCount` URLs have been learned.

```go
type InsufficientDataError struct {
    Count int // Current number of URLs learned
}
```

### `(*Classifier) Stats() Stats`

Returns aggregate statistics about the classifier's current state. Thread-safe.

```go
type Stats struct {
    LearnedCount   int   // Total URLs learned
    NodeCount      int   // Total nodes in the trie
    MaxDepth       int   // Maximum depth of the trie
    MemoryEstimate int64 // Estimated memory usage in bytes
    UniqueValues   int   // Total unique values across all nodes
    PrunedNodes    int   // Nodes with values cleared (high cardinality confirmed)
    CollapsedNodes int   // Nodes with children collapsed to wildcard
}
```

### `(*Classifier) LearnedCount() int`

Returns the number of URLs that have been learned. Thread-safe.

### `(*Classifier) NodeCount() int`

Returns the total number of nodes in the trie. Thread-safe.

## Thread Safety

The classifier is safe for concurrent use:

- `Learn()` and `Classify()` use `sync.RWMutex` for synchronization
- Multiple goroutines can call `Classify()` concurrently (read lock)
- `Learn()` and `Classify()` during learning phase use write locks
- Safe to mix `Learn()` and `Classify()` calls from different goroutines

```go
var wg sync.WaitGroup
c := classifier.NewClassifier(classifier.WithMinLearningCount(100))

// Concurrent classification
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        c.Classify(fmt.Sprintf("/users/%d/profile", id))
    }(i)
}
wg.Wait()
```

## Monitoring

Use the `Stats()` method to monitor classifier memory usage and health:

```go
c := classifier.NewClassifier(
    classifier.WithMinLearningCount(1000),
    classifier.WithMaxValuesPerNode(100),
    classifier.WithPruneHighCardinality(true),
)

// After processing URLs...
stats := c.Stats()
fmt.Printf("URLs learned: %d\n", stats.LearnedCount)
fmt.Printf("Trie nodes: %d (max depth: %d)\n", stats.NodeCount, stats.MaxDepth)
fmt.Printf("Memory estimate: %d bytes\n", stats.MemoryEstimate)
fmt.Printf("Pruned nodes: %d, Collapsed nodes: %d\n", stats.PrunedNodes, stats.CollapsedNodes)
```

## Performance

- **Space**: O(N × M) where N = number of unique URLs, M = average URL segments
- **Time**:
  - Learning: O(N × M) for N URLs
  - Classification: O(M) for M segments

## Contributing

Contributions welcome! Please open an issue or PR.

## License

MIT License
