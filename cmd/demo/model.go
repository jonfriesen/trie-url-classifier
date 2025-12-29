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

// urlPattern pairs an original URL with its normalized pattern and lookup time
type urlPattern struct {
	original   string
	pattern    string
	lookupTime time.Duration
}

type model struct {
	classifier      *classifier.Classifier
	generator       *URLGenerator
	stats           classifier.Stats
	recentPairs     []urlPattern // original URL + normalized pattern
	patternCounts   map[string]int
	totalURLs       int
	startTime       time.Time
	lastTick        time.Time
	urlsLastTick    int
	urlsPerSec      float64
	running         bool
	quitting        bool
	width           int
	height          int
	csvPaths        []string // paths loaded from CSV
	csvIndex        int      // current position in csvPaths
	csvExhausted    bool     // true once we've gone through all CSV paths
	totalLookupTime time.Duration
	lookupCount     int64
}

func newModel(csvPaths []string) model {
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
		csvPaths:      csvPaths,
		csvIndex:      0,
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
			return newModel(nil), tickCmd()
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
		var urls []string
		if len(m.csvPaths) > 0 {
			if !m.csvExhausted {
				// First pass: iterate through CSV in order
				end := m.csvIndex + batchSize
				if end > len(m.csvPaths) {
					end = len(m.csvPaths)
				}
				if m.csvIndex < len(m.csvPaths) {
					urls = m.csvPaths[m.csvIndex:end]
					m.csvIndex = end
				}
				if m.csvIndex >= len(m.csvPaths) {
					m.csvExhausted = true
				}
			} else {
				// After exhaustion: pick random paths from CSV
				urls = make([]string, batchSize)
				for i := 0; i < batchSize; i++ {
					urls[i] = m.csvPaths[m.generator.rng.Intn(len(m.csvPaths))]
				}
			}
		} else {
			// Generate random URLs
			urls = m.generator.GenerateBatch(batchSize)
		}

		for _, url := range urls {
			start := time.Now()
			pattern, err := m.classifier.Classify(url)
			elapsed := time.Since(start)

			m.totalURLs++
			m.totalLookupTime += elapsed
			m.lookupCount++

			if err == nil && pattern != "" {
				m.patternCounts[pattern]++
				m.addRecentPair(url, pattern, elapsed)
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

func (m *model) addRecentPair(url, pattern string, lookupTime time.Duration) {
	// Add to front, keep limited size (allow duplicate patterns with different URLs)
	pair := urlPattern{original: url, pattern: pattern, lookupTime: lookupTime}
	m.recentPairs = append([]urlPattern{pair}, m.recentPairs...)
	if len(m.recentPairs) > maxRecentCount {
		m.recentPairs = m.recentPairs[:maxRecentCount]
	}
}

func (m model) avgLookupTime() time.Duration {
	if m.lookupCount == 0 {
		return 0
	}
	return m.totalLookupTime / time.Duration(m.lookupCount)
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
