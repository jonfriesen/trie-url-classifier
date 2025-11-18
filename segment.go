package classifier

type Segment struct {
	value      string
	children   map[string]*Segment
	isEnd      bool
	values     map[string]int
	totalCount int
}

func NewSegment(value string) *Segment {
	return &Segment{
		value:    value,
		children: make(map[string]*Segment),
		values:   make(map[string]int),
	}
}

func (s *Segment) Cardinality() float64 {
	if s.totalCount == 0 {
		return 0
	}
	return float64(len(s.values)) / float64(s.totalCount)
}

func (s *Segment) IsHighCardinality(threshold float64) bool {
	return s.Cardinality() >= threshold
}
