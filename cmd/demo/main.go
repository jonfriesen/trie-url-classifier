package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	learnFile := flag.String("learn", "", "CSV file with paths to learn (single column)")
	flag.Parse()

	var learnPaths []string
	if *learnFile != "" {
		paths, err := loadPathsFromCSV(*learnFile)
		if err != nil {
			fmt.Printf("Error loading learn file: %v\n", err)
			os.Exit(1)
		}
		learnPaths = paths
		fmt.Printf("Loaded %d paths for learning\n", len(learnPaths))
	}

	p := tea.NewProgram(newModel(learnPaths), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running demo: %v\n", err)
		os.Exit(1)
	}
}

func loadPathsFromCSV(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Strip UTF-8 BOM if present
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	var paths []string

	// Skip header row
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) > 0 {
			path := strings.TrimSpace(record[0])
			if path != "" {
				paths = append(paths, path)
			}
		}
	}

	return paths, nil
}
