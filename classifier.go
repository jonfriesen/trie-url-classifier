package classifier

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type Config struct {
	CardinalityThreshold float64
	MinSamples           int
	MinLearningCount     int
	MaxValuesPerNode     int  // Max unique values to track per node (0 = unlimited)
	PruneHighCardinality bool // Collapse high-cardinality children to bound memory
}

func DefaultConfig() *Config {
	return &Config{
		CardinalityThreshold: 0.75,
		MinSamples:           2,
		MinLearningCount:     0,
		MaxValuesPerNode:     0, // unlimited by default for backwards compatibility
		PruneHighCardinality: false,
	}
}

type Option func(*Config)

func WithCardinalityThreshold(threshold float64) Option {
	return func(c *Config) {
		c.CardinalityThreshold = threshold
	}
}

func WithMinSamples(min int) Option {
	return func(c *Config) {
		c.MinSamples = min
	}
}

func WithMinLearningCount(count int) Option {
	return func(c *Config) {
		c.MinLearningCount = count
	}
}

// WithMaxValuesPerNode limits unique values tracked per trie node.
// Once limit is reached, totalCount keeps incrementing but no new values are stored.
// This bounds memory usage for long-running classifiers. Use 0 for unlimited.
func WithMaxValuesPerNode(max int) Option {
	return func(c *Config) {
		c.MaxValuesPerNode = max
	}
}

// WithPruneHighCardinality clears the values map once a node is confirmed
// as high cardinality, saving additional memory. The node retains its
// totalCount for cardinality estimation.
func WithPruneHighCardinality(prune bool) Option {
	return func(c *Config) {
		c.PruneHighCardinality = prune
	}
}

type Classifier struct {
	root         *Segment
	config       *Config
	mu           sync.RWMutex
	learnedCount int
}

func NewClassifier(opts ...Option) *Classifier {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	return &Classifier{
		root:   NewSegment(""),
		config: config,
	}
}

func (c *Classifier) Learn(urls []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, url := range urls {
		c.insert(url)
		c.learnedCount++
	}
}

func (c *Classifier) insert(url string) {
	if url == "" {
		return
	}

	parts := c.splitURL(url)
	node := c.root

	for _, part := range parts {
		var child *Segment

		// If parent is collapsed, route through wildcard child
		if node.collapsed {
			if node.children["*"] == nil {
				node.children["*"] = NewSegment("*")
			}
			child = node.children["*"]
		} else {
			if node.children[part] == nil {
				node.children[part] = NewSegment(part)
			}
			child = node.children[part]
		}

		child.totalCount++

		// Only track value if below max limit (0 = unlimited)
		if c.config.MaxValuesPerNode == 0 || len(child.values) < c.config.MaxValuesPerNode {
			child.values[part]++
		} else if _, exists := child.values[part]; exists {
			child.values[part]++
		}

		// Check if we should collapse this node's children (memory optimization)
		// Only collapse when children look like dynamic parameters (UUIDs, IDs, etc.)
		// not when they're static path segments like "api", "users", etc.
		if c.config.PruneHighCardinality && !node.collapsed &&
			len(node.children) >= c.config.MaxValuesPerNode &&
			c.hasHighVariability(node) && c.childrenLookDynamic(node) {
			c.collapseChildren(node)
		}

		node = child
	}

	node.isEnd = true
}

// childrenLookDynamic checks if the majority of a node's children
// appear to be dynamic values (UUIDs, IDs, etc.) rather than static paths
func (c *Classifier) childrenLookDynamic(node *Segment) bool {
	if len(node.children) == 0 {
		return false
	}

	dynamicCount := 0
	for childName := range node.children {
		if c.looksLikeParameter(childName) {
			dynamicCount++
		}
	}

	// Require majority of children to look dynamic
	return float64(dynamicCount)/float64(len(node.children)) >= 0.5
}

// collapseChildren merges all children into a single wildcard child
func (c *Classifier) collapseChildren(node *Segment) {
	if node.collapsed || len(node.children) == 0 {
		return
	}

	// Create or get wildcard child
	wildcard := NewSegment("*")
	wildcard.pruned = true

	// Merge all children's stats and grandchildren into wildcard
	for _, child := range node.children {
		wildcard.totalCount += child.totalCount
		if child.isEnd {
			wildcard.isEnd = true
		}
		// Merge grandchildren
		for name, grandchild := range child.children {
			if wildcard.children[name] == nil {
				wildcard.children[name] = grandchild
			} else {
				// Merge stats
				wildcard.children[name].totalCount += grandchild.totalCount
				for v, cnt := range grandchild.values {
					wildcard.children[name].values[v] += cnt
				}
			}
		}
	}

	// Replace all children with single wildcard
	node.children = map[string]*Segment{"*": wildcard}
	node.collapsed = true
}

func (c *Classifier) Classify(url string) (string, error) {
	if url == "" {
		return "", nil
	}

	// Always learn during Classify (memory is bounded by PruneHighCardinality)
	c.mu.Lock()
	c.insert(url)
	c.learnedCount++
	count := c.learnedCount
	belowMin := c.config.MinLearningCount > 0 && count <= c.config.MinLearningCount
	c.mu.Unlock()

	// Return error if still in learning phase
	if belowMin {
		return "", &InsufficientDataError{Count: count}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	parts := c.splitURL(url)
	if len(parts) == 0 {
		return "/", nil
	}

	normalized := make([]string, 0, len(parts))
	node := c.root

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		// Handle collapsed nodes - they are always high variability
		if node.collapsed {
			paramType := c.classifyParameterType(part)
			normalized = append(normalized, "{"+paramType+"}")

			// Use wildcard child to continue
			if wildcardChild, exists := node.children["*"]; exists {
				node = wildcardChild
			}
			continue
		}

		if child, exists := node.children[part]; exists {
			if c.hasHighVariability(node) {
				paramType := c.classifyParameterType(part)
				normalized = append(normalized, "{"+paramType+"}")

				commonChildren := c.findCommonChildrenAcrossAllSiblings(node)
				if len(commonChildren) > 0 {
					virtualNode := &Segment{
						value:    "",
						children: make(map[string]*Segment),
						isEnd:    false,
					}
					for k, v := range commonChildren {
						virtualNode.children[k] = v
					}
					node = virtualNode
					continue
				}
				node = child
			} else {
				normalized = append(normalized, part)
				node = child
			}
			continue
		}

		if c.hasHighVariability(node) {
			paramType := c.classifyParameterType(part)
			normalized = append(normalized, "{"+paramType+"}")

			commonChildren := c.findCommonChildrenAcrossAllSiblings(node)
			if len(commonChildren) > 0 {
				virtualNode := &Segment{
					value:    "",
					children: make(map[string]*Segment),
					isEnd:    false,
				}
				for k, v := range commonChildren {
					virtualNode.children[k] = v
				}
				node = virtualNode
				continue
			}

			for j := i + 1; j < len(parts); j++ {
				remainingPart := parts[j]
				paramType := c.classifyParameterType(remainingPart)
				normalized = append(normalized, "{"+paramType+"}")
			}
			break
		}

		for j := i; j < len(parts); j++ {
			normalized = append(normalized, parts[j])
		}
		break
	}

	return "/" + strings.Join(normalized, "/"), nil
}

func (c *Classifier) shouldParameterize(segment *Segment) bool {
	if segment.totalCount < c.config.MinSamples {
		return false
	}

	if segment.IsHighCardinality(c.config.CardinalityThreshold) {
		return true
	}

	return false
}

func (c *Classifier) hasHighVariability(node *Segment) bool {
	// Special case: if there's only one child but it's been traversed multiple times
	// and looks like a parameter pattern, treat it as variable
	if len(node.children) == 1 {
		for childValue, child := range node.children {
			if child.totalCount >= c.config.MinSamples && c.looksLikeParameter(childValue) {
				return true
			}
		}
	}

	minChildren := 3
	if c.config.CardinalityThreshold < 0.75 {
		minChildren = 2
	}

	if len(node.children) < minChildren {
		return false
	}

	totalTraversals := 0
	for _, child := range node.children {
		totalTraversals += child.totalCount
	}

	variability := float64(len(node.children)) / float64(totalTraversals)

	return variability >= c.config.CardinalityThreshold
}

func (c *Classifier) findCommonChildrenAcrossAllSiblings(node *Segment) map[string]*Segment {
	if len(node.children) == 0 {
		return nil
	}

	allChildren := make([]*Segment, 0, len(node.children))
	for _, child := range node.children {
		allChildren = append(allChildren, child)
	}

	return c.mergeChildren(allChildren)
}

func (c *Classifier) mergeChildren(segments []*Segment) map[string]*Segment {
	if len(segments) == 0 {
		return nil
	}

	childrenByName := make(map[string][]*Segment)

	for _, segment := range segments {
		for childName, childNode := range segment.children {
			childrenByName[childName] = append(childrenByName[childName], childNode)
		}
	}

	result := make(map[string]*Segment)
	for childName, childNodes := range childrenByName {
		mergedChild := NewSegment(childName)

		for _, childNode := range childNodes {
			for grandchildName, grandchildNode := range childNode.children {
				if mergedChild.children[grandchildName] == nil {
					mergedChild.children[grandchildName] = grandchildNode
				}
			}

			for value, count := range childNode.values {
				mergedChild.values[value] += count
			}
			mergedChild.totalCount += childNode.totalCount
		}

		result[childName] = mergedChild
	}

	return result
}

func (c *Classifier) looksLikeParameter(value string) bool {
	if matched, _ := regexp.MatchString(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, value); matched {
		return true
	}

	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, value); matched {
		return true
	}

	if matched, _ := regexp.MatchString(`^\d{10,}$`, value); matched {
		return true
	}

	if matched, _ := regexp.MatchString(`^[0-9a-f]{24,}$`, value); matched {
		return true
	}

	if matched, _ := regexp.MatchString(`^(cus|sub|prod|price|pm|pi|ch|in|tok|src|ba|card)_[a-zA-Z0-9]+$`, value); matched {
		return true
	}

	if num, err := strconv.ParseInt(value, 10, 64); err == nil {
		if num >= 100 && num < 2000 {
			return true
		}
		if num >= 2100 && num < 10000 {
			return true
		}
		if num >= 100000 {
			return true
		}
		return false
	}

	// Slug pattern with specific characteristics that suggest it's a dynamic value
	// Must contain at least one hyphen AND either:
	// - ends with digits
	// - has multiple segments
	if matched, _ := regexp.MatchString(`^[a-z0-9]+-[a-z0-9-]+-\d+$`, value); matched {
		return true // Slug ending with numeric ID (e.g., "my-post-12345")
	}

	return false
}

func (c *Classifier) classifyParameterType(value string) string {
	if matched, _ := regexp.MatchString(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, value); matched {
		return "uuid"
	}

	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, value); matched {
		return "date"
	}

	if matched, _ := regexp.MatchString(`^\d{10,}$`, value); matched {
		return "timestamp"
	}

	if matched, _ := regexp.MatchString(`^[0-9a-f]{24,}$`, value); matched {
		return "hash"
	}

	if matched, _ := regexp.MatchString(`^(cus|sub|prod|price|pm|pi|ch|in|tok|src|ba|card)_[a-zA-Z0-9]+$`, value); matched {
		return "id"
	}

	if num, err := strconv.ParseInt(value, 10, 64); err == nil {
		if num >= 100 && num < 10000 {
			return "id"
		}
		if num >= 100000 {
			return "id"
		}
	}

	if matched, _ := regexp.MatchString(`^[a-z0-9]+(-[a-z0-9]+)*(-\d+)?$`, value); matched {
		return "slug"
	}

	return "param"
}

func (c *Classifier) splitURL(url string) []string {
	url = strings.TrimPrefix(url, "/")

	if url == "" {
		return []string{}
	}

	return strings.Split(url, "/")
}
