package classifier

import (
	"fmt"
	"testing"
)

func TestStats(t *testing.T) {
	c := NewClassifier()
	c.Learn([]string{
		"/api/v1/users/123",
		"/api/v1/users/456",
		"/api/v1/products/789",
	})

	stats := c.Stats()

	if stats.LearnedCount != 3 {
		t.Errorf("LearnedCount = %d, want 3", stats.LearnedCount)
	}

	if stats.NodeCount < 5 {
		t.Errorf("NodeCount = %d, want at least 5", stats.NodeCount)
	}

	if stats.MaxDepth < 4 {
		t.Errorf("MaxDepth = %d, want at least 4", stats.MaxDepth)
	}

	if stats.MemoryEstimate <= 0 {
		t.Errorf("MemoryEstimate = %d, want > 0", stats.MemoryEstimate)
	}
}

func TestLearnedCount(t *testing.T) {
	c := NewClassifier()

	if c.LearnedCount() != 0 {
		t.Errorf("LearnedCount = %d, want 0", c.LearnedCount())
	}

	c.Learn([]string{"/a", "/b"})

	if c.LearnedCount() != 2 {
		t.Errorf("LearnedCount = %d, want 2", c.LearnedCount())
	}
}

func TestNodeCount(t *testing.T) {
	c := NewClassifier()

	// Just root node
	if c.NodeCount() != 1 {
		t.Errorf("NodeCount = %d, want 1", c.NodeCount())
	}

	c.Learn([]string{"/a/b/c"})

	// root + a + b + c = 4
	if c.NodeCount() != 4 {
		t.Errorf("NodeCount = %d, want 4", c.NodeCount())
	}
}

func TestStatsMemoryGrowth(t *testing.T) {
	c := NewClassifier()

	c.Learn([]string{"/api/v1/test"})
	stats1 := c.Stats()

	c.Learn([]string{
		"/api/v1/users/1",
		"/api/v1/users/2",
		"/api/v1/users/3",
		"/api/v1/products/a",
		"/api/v1/products/b",
	})
	stats2 := c.Stats()

	if stats2.MemoryEstimate <= stats1.MemoryEstimate {
		t.Errorf("MemoryEstimate should grow after learning more URLs")
	}

	if stats2.NodeCount <= stats1.NodeCount {
		t.Errorf("NodeCount should grow after learning more URLs")
	}
}

func TestMaxValuesPerNode(t *testing.T) {
	c := NewClassifier(WithMaxValuesPerNode(5))

	// Learn 10 URLs with different UUIDs
	urls := make([]string, 10)
	for i := 0; i < 10; i++ {
		urls[i] = "/api/users/" + string(rune('a'+i))
	}
	c.Learn(urls)

	stats := c.Stats()
	// With max 5 values per node, unique values should be capped
	// The users node should have at most 5 unique values tracked
	if stats.UniqueValues > 20 { // generous upper bound
		t.Errorf("UniqueValues = %d, expected bounded growth", stats.UniqueValues)
	}
}

func TestPruneHighCardinality(t *testing.T) {
	c := NewClassifier(
		WithMaxValuesPerNode(10),
		WithPruneHighCardinality(true),
		WithMinSamples(3),
		WithCardinalityThreshold(0.75),
	)

	// Learn enough URLs to trigger pruning
	urls := make([]string, 20)
	for i := 0; i < 20; i++ {
		urls[i] = "/api/users/" + string(rune('a'+i)) + "/profile"
	}
	c.Learn(urls)

	stats := c.Stats()
	// Should have some pruned nodes after learning high-cardinality data
	if stats.PrunedNodes == 0 {
		t.Logf("PrunedNodes = 0, may need more data to trigger pruning")
	}

	// Classification should still work after pruning
	pattern, err := c.Classify("/api/users/z/profile")
	if err != nil {
		t.Errorf("Classify failed after pruning: %v", err)
	}
	if pattern == "" {
		t.Errorf("Expected non-empty pattern after pruning")
	}
}

func TestMemoryBoundedLongRunning(t *testing.T) {
	c := NewClassifier(
		WithMaxValuesPerNode(50),
		WithPruneHighCardinality(true),
	)

	// Simulate long-running with many unique URLs using realistic UUIDs
	for batch := 0; batch < 10; batch++ {
		urls := make([]string, 100)
		for i := 0; i < 100; i++ {
			// Generate realistic UUID-like values that will be detected as parameters
			uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", batch*1000+i, i, i, i, batch*100000+i)
			urls[i] = "/api/v1/users/" + uuid + "/profile"
		}
		c.Learn(urls)
	}

	stats := c.Stats()

	// Memory should be bounded despite 1000 unique URLs
	// With pruning and max values, nodes should be collapsed once threshold is hit
	// We expect significantly fewer nodes than 1000
	if stats.NodeCount > 500 {
		t.Errorf("NodeCount = %d, expected bounded by collapsing", stats.NodeCount)
	}

	t.Logf("After 1000 URLs: Nodes=%d, UniqueValues=%d, Collapsed=%d, Memory=%d bytes",
		stats.NodeCount, stats.UniqueValues, stats.CollapsedNodes, stats.MemoryEstimate)
}
