package classifier

import (
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	CardinalityThreshold float64
	MinSamples           int
}

func DefaultConfig() *Config {
	return &Config{
		CardinalityThreshold: 0.75,
		MinSamples:           2,
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

type Classifier struct {
	root   *Segment
	config *Config
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
	for _, url := range urls {
		c.insert(url)
	}
}

func (c *Classifier) insert(url string) {
	if url == "" {
		return
	}

	parts := c.splitURL(url)
	node := c.root

	for _, part := range parts {
		if node.children[part] == nil {
			node.children[part] = NewSegment(part)
		}

		child := node.children[part]
		child.values[part]++
		child.totalCount++

		node = child
	}

	node.isEnd = true
}

func (c *Classifier) Classify(url string) string {
	if url == "" {
		return ""
	}

	parts := c.splitURL(url)
	if len(parts) == 0 {
		return "/"
	}

	normalized := make([]string, 0, len(parts))
	node := c.root

	for i := 0; i < len(parts); i++ {
		part := parts[i]

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

	return "/" + strings.Join(normalized, "/")
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
