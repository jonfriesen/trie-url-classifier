package main

import (
	"fmt"
	"time"

	classifier "github.com/jonfriesen/trie-url-classifier"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	batchSize      = 1000
	tickInterval   = 100 * time.Millisecond
	maxRecentCount = 15
)

type tickMsg time.Time

// urlPattern pairs an original URL with its normalized pattern
type urlPattern struct {
	original string
	pattern  string
}

type model struct {
	classifier    *classifier.Classifier
	generator     *URLGenerator
	stats         classifier.Stats
	recentPairs   []urlPattern // original URL + normalized pattern
	patternCounts map[string]int
	totalURLs     int
	startTime     time.Time
	lastTick      time.Time
	urlsLastTick  int
	urlsPerSec    float64
	running       bool
	quitting      bool
	width         int
	height        int
}

func newModel() model {
	c := classifier.NewClassifier(
		classifier.WithMinLearningCount(100),
		classifier.WithCardinalityThreshold(0.75),
		classifier.WithMinSamples(2),
		classifier.WithMaxValuesPerNode(100),      // Cap unique values per node
		classifier.WithPruneHighCardinality(true), // Collapse high-cardinality nodes to bound memory
	)

	return model{
		classifier:    c,
		generator:     NewURLGenerator(time.Now().UnixNano()),
		recentPairs:   make([]urlPattern, 0, maxRecentCount),
		patternCounts: make(map[string]int),
		startTime:     time.Now(),
		lastTick:      time.Now(),
		running:       true,
		width:         120,
		height:        24,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), tea.EnterAltScreen)
}

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case " ":
			m.running = !m.running
			return m, nil
		case "r":
			return newModel(), tickCmd()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if !m.running {
			return m, tickCmd()
		}

		// Process a batch of URLs (Classify auto-learns, memory bounded by pruning)
		urls := m.generator.GenerateBatch(batchSize)
		for _, url := range urls {
			pattern, err := m.classifier.Classify(url)
			m.totalURLs++
			if err == nil && pattern != "" {
				m.patternCounts[pattern]++
				m.addRecentPair(url, pattern)
			}
		}

		// Calculate URLs/sec
		now := time.Now()
		elapsed := now.Sub(m.lastTick).Seconds()
		if elapsed > 0 {
			urlsProcessed := m.totalURLs - m.urlsLastTick
			m.urlsPerSec = float64(urlsProcessed) / elapsed
		}
		m.lastTick = now
		m.urlsLastTick = m.totalURLs

		// Update stats
		m.stats = m.classifier.Stats()

		return m, tickCmd()
	}

	return m, nil
}

func (m *model) addRecentPair(url, pattern string) {
	// Add to front, keep limited size (allow duplicate patterns with different URLs)
	pair := urlPattern{original: url, pattern: pattern}
	m.recentPairs = append([]urlPattern{pair}, m.recentPairs...)
	if len(m.recentPairs) > maxRecentCount {
		m.recentPairs = m.recentPairs[:maxRecentCount]
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.2fM", float64(n)/1000000)
}
