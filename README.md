# Trie-Based URL Pattern Classifier

A Go library that automatically learns URL patterns from batches of URLs and normalizes them by detecting dynamic segments.

## Why This Library?

We built this library to normalize URLs at [Zori](https://zorihq.com) when processing customer analytics events. High-cardinality URLs with dynamic IDs, UUIDs, and dates create storage and query performance issues that this library solves by:

- **Automatic pattern detection** - No manual route definitions needed
- **Batch-based learning** - Discovers patterns from actual URLs in each batch
- **Cardinality reduction** - Converts `/users/12345/profile` to `/users/{id}/profile`
- **Type-aware** - Distinguishes UUIDs, IuDs, dates, hashes, and other parameter types

Works for analytics, metrics storage, log normalization, and any system that needs to aggregate URL-based data.

## Key Features

- **Batch-Based Processing**: Create a new classifier instance per batch to avoid memory growth
- **Smart Pattern Detection**: Detects high-cardinality segments automatically
- **Parameter Type Classification**: Identifies UUIDs, IDs, dates, timestamps, hashes, tokens, and slugs
- **No Upfront Knowledge**: Learns patterns from actual URLs in each batch
- **Configurable**: Adjust cardinality thresholds and minimum sample requirements

## Installation

```bash
go get github.com/ZoriHQ/trie-url-classifier
```

## Quick Start

```go
package main

import "fmt"

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
    classifier := NewClassifier()

    // Learn patterns from the batch
    classifier.Learn(batch)

    // Normalize URLs in the batch
    for _, url := range batch {
        pattern := classifier.Classify(url)
        fmt.Printf("%s -> %s\n", url, pattern)
    }

    // After processing, discard the classifier
    // For the next batch, create a NEW instance
}
```

## Usage Pattern

Process batches of URLs independently:

```go
func processBatch(urlBatch []string) {
    classifier := NewClassifier()
    classifier.Learn(urlBatch)

    for _, url := range urlBatch {
        normalizedURL := classifier.Classify(url)
        // Use normalizedURL for storage, aggregation, etc.
    }
}
```

**Important**: Create a new classifier for each batch to avoid memory growth.

## Configuration

Customize the classifier behavior:

```go
classifier := NewClassifier(
    WithCardinalityThreshold(0.75),  // 75% unique values = dynamic (default)
    WithMinSamples(2),                // Minimum samples before detection (default: 2)
)
```

### Configuration Options

- **CardinalityThreshold** (default: 0.75): Ratio of unique values to total count. Higher = stricter detection
- **MinSamples** (default: 2): Minimum samples needed at a position before considering it for parametrization

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

Learns patterns from a batch of URLs. Can be called multiple times.

### `(*Classifier) Classify(url string) string`

Normalizes a URL based on learned patterns.

### Configuration Options

- `WithCardinalityThreshold(threshold float64) Option`
- `WithMinSamples(min int) Option`

## How Pattern Detection Works

The classifier detects dynamic segments through:

1. **Cardinality analysis** - Segments with many unique values (≥75% by default) are marked as dynamic
2. **Pattern matching** - Even with identical values, segments matching UUID/ID/date patterns are parameterized
3. **Static preservation** - Common words like "api", "users", "settings" remain static

This handles both diverse batches and edge cases like repeated identical URLs.

## Performance

- **Space**: O(N × M) where N = number of unique URLs, M = average URL segments
- **Time**:
  - Learning: O(N × M) for N URLs
  - Classification: O(M) for M segments

## Contributing

Contributions welcome! Please open an issue or PR.

## License

MIT License
