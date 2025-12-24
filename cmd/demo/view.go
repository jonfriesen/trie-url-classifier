package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Bold(true)

	urlStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	patternStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117"))

	paramStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	arrowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	statusRunning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	statusPaused = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// Title bar with status
	status := statusRunning.Render("RUNNING")
	if !m.running {
		status = statusPaused.Render("PAUSED")
	}
	elapsed := time.Since(m.startTime).Round(time.Second)
	titleBar := fmt.Sprintf("%s  %s  %s  %s",
		titleStyle.Render("URL Classifier Demo"),
		status,
		labelStyle.Render("Uptime:")+valueStyle.Render(elapsed.String()),
		helpStyle.Render("[q]uit [space]pause [r]eset"))

	// Top row: Stats + Distribution side by side
	statsContent := m.renderStatsCompact()
	leftBox := boxStyle.Width(40).Height(8).Render(statsContent)

	distContent := m.renderDistributionCompact()
	rightBox := boxStyle.Width(36).Height(8).Render(distContent)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)

	// Bottom row: Recent URL -> Pattern mappings (full width)
	patternsContent := m.renderRecentPairs()
	bottomBox := boxStyle.Width(78).Render(patternsContent)

	return titleBar + "\n" + topRow + "\n" + bottomBox + "\n"
}

func (m model) renderStatsCompact() string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Statistics"))
	sb.WriteString("\n\n")

	stats := [][]string{
		{fmt.Sprintf("URLs: %s", valueStyle.Render(formatNumber(m.stats.LearnedCount))),
			fmt.Sprintf("Rate: %s/s", valueStyle.Render(fmt.Sprintf("%.0f", m.urlsPerSec)))},
		{fmt.Sprintf("Nodes: %s", valueStyle.Render(formatNumber(m.stats.NodeCount))),
			fmt.Sprintf("Depth: %s", valueStyle.Render(fmt.Sprintf("%d", m.stats.MaxDepth)))},
		{fmt.Sprintf("Memory: %s", valueStyle.Render(formatBytes(m.stats.MemoryEstimate))),
			fmt.Sprintf("Collapsed: %s", valueStyle.Render(formatNumber(m.stats.CollapsedNodes)))},
		{fmt.Sprintf("Patterns: %s", valueStyle.Render(formatNumber(len(m.patternCounts)))), ""},
	}

	for _, row := range stats {
		left := labelStyle.Width(20).Render(row[0])
		right := labelStyle.Render(row[1])
		sb.WriteString(left + right + "\n")
	}

	return sb.String()
}

func (m model) renderDistributionCompact() string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Param Types"))
	sb.WriteString("\n\n")

	if len(m.patternCounts) == 0 {
		sb.WriteString(labelStyle.Render("(learning...)"))
		return sb.String()
	}

	typeCounts := make(map[string]int)
	for pattern, count := range m.patternCounts {
		parts := strings.Split(pattern, "/")
		for _, part := range parts {
			if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
				paramType := part[1 : len(part)-1]
				typeCounts[paramType] += count
			}
		}
	}

	type typeCount struct {
		name  string
		count int
	}
	var sorted []typeCount
	total := 0
	for name, count := range typeCounts {
		sorted = append(sorted, typeCount{name, count})
		total += count
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	for i, tc := range sorted {
		if i >= 5 {
			break
		}
		pct := float64(tc.count) / float64(total) * 100
		barLen := int(pct / 10)
		if barLen < 1 && pct > 0 {
			barLen = 1
		}
		bar := strings.Repeat("█", barLen)
		line := fmt.Sprintf("%s %s %s",
			paramStyle.Width(10).Render("{"+tc.name+"}"),
			labelStyle.Width(10).Render(bar),
			valueStyle.Render(fmt.Sprintf("%4.1f%%", pct)))
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func (m model) renderRecentPairs() string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Recent URLs"))
	sb.WriteString(fmt.Sprintf(" %s\n\n", labelStyle.Render(fmt.Sprintf("(%d unique patterns)", len(m.patternCounts)))))

	if len(m.recentPairs) == 0 {
		sb.WriteString(labelStyle.Render("(learning...)"))
		return sb.String()
	}

	arrow := arrowStyle.Render(" → ")

	for i, pair := range m.recentPairs {
		if i >= 8 {
			break
		}
		original := urlStyle.Render(truncate(pair.original, 35))
		pattern := highlightParams(pair.pattern)
		sb.WriteString(fmt.Sprintf("%-38s%s%s\n", original, arrow, pattern))
	}

	return sb.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func highlightParams(pattern string) string {
	parts := strings.Split(pattern, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			parts[i] = paramStyle.Render(part)
		} else {
			parts[i] = patternStyle.Render(part)
		}
	}
	return strings.Join(parts, patternStyle.Render("/"))
}
