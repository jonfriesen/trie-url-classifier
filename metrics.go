package classifier

// Stats contains aggregate statistics about the classifier state.
type Stats struct {
	LearnedCount   int   // Total URLs learned
	NodeCount      int   // Total nodes in the trie
	MaxDepth       int   // Maximum depth of the trie
	MemoryEstimate int64 // Estimated memory usage in bytes
	UniqueValues   int   // Total unique values across all nodes
	PrunedNodes    int   // Nodes with values cleared (high cardinality confirmed)
	CollapsedNodes int   // Nodes with children collapsed to wildcard
}

// Stats returns aggregate statistics about the classifier's current state.
func (c *Classifier) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := Stats{
		LearnedCount: c.learnedCount,
	}

	c.traverseForStats(c.root, 0, &stats)
	return stats
}

// LearnedCount returns the number of URLs that have been learned.
func (c *Classifier) LearnedCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.learnedCount
}

// NodeCount returns the total number of nodes in the trie.
func (c *Classifier) NodeCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.countNodes(c.root)
}

func (c *Classifier) countNodes(node *Segment) int {
	if node == nil {
		return 0
	}
	count := 1
	for _, child := range node.children {
		count += c.countNodes(child)
	}
	return count
}

func (c *Classifier) traverseForStats(node *Segment, depth int, stats *Stats) {
	if node == nil {
		return
	}

	stats.NodeCount++
	if depth > stats.MaxDepth {
		stats.MaxDepth = depth
	}

	// Count pruned and collapsed nodes
	if node.pruned {
		stats.PrunedNodes++
	}
	if node.collapsed {
		stats.CollapsedNodes++
	}

	// Count unique values in this node
	stats.UniqueValues += len(node.values)

	// Estimate memory for this node:
	// - Segment struct overhead: ~96 bytes (added pruned bool, uniqueCount int)
	// - children map: 8 bytes per entry (pointer)
	// - values map: ~24 bytes per entry (string key avg 16 bytes + int 8 bytes)
	// - value string: avg 16 bytes
	stats.MemoryEstimate += 96
	stats.MemoryEstimate += int64(len(node.children) * 8)
	stats.MemoryEstimate += int64(len(node.values) * 24)
	stats.MemoryEstimate += int64(len(node.value))

	for _, child := range node.children {
		c.traverseForStats(child, depth+1, stats)
	}
}
