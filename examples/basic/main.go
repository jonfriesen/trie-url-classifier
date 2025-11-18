package main

import (
	"fmt"

	urlClassifier "github.com/ZoriHQ/trie-url-classifier"
)

func main() {
	fmt.Println("=== Trie-Based URL Pattern Classifier ===")
	fmt.Println()
	batch := []string{
		"/projects/d381b052-99eb-40f2-9ede-9bce790faae1/analytics",
		"/projects/a1b2c3d4-e5f6-7890-abcd-ef1234567890/analytics",
		"/projects/12345678-1234-1234-1234-123456789012/analytics",
		"/projects/f47ac10b-58cc-4372-a567-0e02b2c3d479/settings",
		"/projects/550e8400-e29b-41d4-a716-446655440000/settings",
		"/users/123456/profile",
		"/users/789012/profile",
		"/users/345678/profile",
		"/users/901234/settings",
		"/api/v1/health",
		"/api/v1/health",
		"/api/v1/status",
		"/reports/2024-01-15/summary",
		"/reports/2024-01-16/summary",
		"/reports/2024-01-17/summary",
		"/products/507f1f77bcf86cd799439011/details",
		"/products/507f191e810c19729de860ea/details",
		"/products/507f1f77bcf86cd799439999/details",
	}

	classifier := urlClassifier.NewClassifier()

	fmt.Println("Processing batch of", len(batch), "URLs...")
	fmt.Println()

	classifier.Learn(batch)

	testURLs := append(batch[:3],
		"/projects/ffffffff-ffff-ffff-ffff-ffffffffffff/analytics",
		"/users/999999/profile",
		"/reports/2024-12-25/summary",
	)

	fmt.Println("Classified URLs:")
	fmt.Println("-----------------------------------")
	for _, url := range testURLs {
		pattern := classifier.Classify(url)
		fmt.Printf("%-70s -> %s\n", url, pattern)
	}

	fmt.Println()
	fmt.Println("=== Example: Processing Multiple Batches ===")
	fmt.Println()

	batches := [][]string{
		{
			"/orders/12345/details",
			"/orders/67890/details",
			"/orders/11111/details",
		},
		{
			"/customers/cus_abc123/profile",
			"/customers/cus_def456/profile",
			"/customers/cus_xyz789/profile",
		},
	}

	for i, batchData := range batches {
		fmt.Printf("Batch %d:\n", i+1)

		batchClassifier := urlClassifier.NewClassifier()
		batchClassifier.Learn(batchData)

		for _, url := range batchData {
			pattern := batchClassifier.Classify(url)
			fmt.Printf("  %s -> %s\n", url, pattern)
		}

		fmt.Println()
	}
}
