package classifier

type Segment struct {
	value       string
	children    map[string]*Segment
	isEnd       bool
	values      map[string]int
	totalCount  int
	pruned      bool // true if values map was cleared after confirming high cardinality
	uniqueCount int  // preserved count of unique values when pruned
	collapsed   bool // true if children were collapsed into wildcard (memory optimization)
}

func NewSegment(value string) *Segment {
	return &Segment{
		value:    value,
		children: make(map[string]*Segment),
		values:   make(map[string]int),
	}
}

// Cardinality returns the ratio of unique values to total occurrences.
// For pruned nodes, returns 1.0 (confirmed high cardinality).
// For capped nodes, uses the capped unique count.
func (s *Segment) Cardinality() float64 {
	if s.totalCount == 0 {
		return 0
	}
	if s.pruned {
		return 1.0 // confirmed high cardinality
	}
	return float64(len(s.values)) / float64(s.totalCount)
}

func (s *Segment) IsHighCardinality(threshold float64) bool {
	return s.Cardinality() >= threshold
}

// IsPruned returns true if this segment's values were cleared
// after confirming high cardinality.
func (s *Segment) IsPruned() bool {
	return s.pruned
}
