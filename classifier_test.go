package classifier

import (
	"testing"
)

func TestClassifier_SingleURLs(t *testing.T) {
	tests := []struct {
		name         string
		trainingURLs []string
		testURL      string
		expected     string
	}{
		{
			name:         "simple static path",
			trainingURLs: []string{"/about", "/about", "/about"},
			testURL:      "/about",
			expected:     "/about",
		},
		{
			name:         "static nested path",
			trainingURLs: []string{"/api/v1/health", "/api/v1/health", "/api/v1/health"},
			testURL:      "/api/v1/health",
			expected:     "/api/v1/health",
		},
		{
			name: "path with UUID",
			trainingURLs: []string{
				"/projects/d381b052-99eb-40f2-9ede-9bce790faae1/analytics",
				"/projects/a1b2c3d4-e5f6-7890-abcd-ef1234567890/analytics",
				"/projects/12345678-1234-1234-1234-123456789012/analytics",
			},
			testURL:  "/projects/d381b052-99eb-40f2-9ede-9bce790faae1/analytics",
			expected: "/projects/{uuid}/analytics",
		},
		{
			name: "path with UUID all the same",
			trainingURLs: []string{
				"/projects/a1b2c3d4-e5f6-7890-abcd-ef1234567890/analytics",
				"/projects/a1b2c3d4-e5f6-7890-abcd-ef1234567890/analytics",
				"/projects/a1b2c3d4-e5f6-7890-abcd-ef1234567890/analytics",
			},
			testURL:  "/projects/a1b2c3d4-e5f6-7890-abcd-ef1234567890/analytics",
			expected: "/projects/{uuid}/analytics",
		},
		{
			name: "path with multiple UUIDs",
			trainingURLs: []string{
				"/orgs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/projects/d381b052-99eb-40f2-9ede-9bce790faae1",
				"/orgs/11111111-1111-1111-1111-111111111111/projects/22222222-2222-2222-2222-222222222222",
				"/orgs/33333333-3333-3333-3333-333333333333/projects/44444444-4444-4444-4444-444444444444",
			},
			testURL:  "/orgs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/projects/d381b052-99eb-40f2-9ede-9bce790faae1",
			expected: "/orgs/{uuid}/projects/{uuid}",
		},
		{
			name: "path with numeric ID",
			trainingURLs: []string{
				"/users/123456/profile",
				"/users/789012/profile",
				"/users/345678/profile",
			},
			testURL:  "/users/123456/profile",
			expected: "/users/{id}/profile",
		},
		{
			name: "path with hash",
			trainingURLs: []string{
				"/products/507f1f77bcf86cd799439011/details",
				"/products/507f191e810c19729de860ea/details",
				"/products/507f1f77bcf86cd799439999/details",
			},
			testURL:  "/products/507f1f77bcf86cd799439011/details",
			expected: "/products/{hash}/details",
		},
		{
			name: "path with date",
			trainingURLs: []string{
				"/reports/2024-01-15/summary",
				"/reports/2024-01-16/summary",
				"/reports/2024-01-17/summary",
			},
			testURL:  "/reports/2024-01-15/summary",
			expected: "/reports/{date}/summary",
		},
		{
			name: "path with timestamp",
			trainingURLs: []string{
				"/events/1705334400/logs",
				"/events/1705334401/logs",
				"/events/1705334402/logs",
			},
			testURL:  "/events/1705334400/logs",
			expected: "/events/{timestamp}/logs",
		},
		{
			name: "path with slug-id combo",
			trainingURLs: []string{
				"/blog/my-awesome-post-12345",
				"/blog/another-great-post-67890",
				"/blog/best-post-ever-11111",
			},
			testURL:  "/blog/my-awesome-post-12345",
			expected: "/blog/{slug}",
		},
		{
			name: "mixed static and dynamic",
			trainingURLs: []string{
				"/api/v2/users/987654/settings/notifications",
				"/api/v2/users/123456/settings/notifications",
				"/api/v2/users/555555/settings/notifications",
			},
			testURL:  "/api/v2/users/987654/settings/notifications",
			expected: "/api/v2/users/{id}/settings/notifications",
		},
		{
			name:         "root path",
			trainingURLs: []string{"/", "/", "/"},
			testURL:      "/",
			expected:     "/",
		},
		{
			name:         "empty path",
			trainingURLs: []string{"", "", ""},
			testURL:      "",
			expected:     "",
		},
		{
			name: "stripe customer ID",
			trainingURLs: []string{
				"/customers/cus_1234567890abcdef",
				"/customers/cus_abcdef1234567890",
				"/customers/cus_xyz789abc123def4",
			},
			testURL:  "/customers/cus_1234567890abcdef",
			expected: "/customers/{id}",
		},
		{
			name: "stripe subscription ID",
			trainingURLs: []string{
				"/subscriptions/sub_abcdef1234567890",
				"/subscriptions/sub_1234567890abcdef",
				"/subscriptions/sub_xyz789abc123def4",
			},
			testURL:  "/subscriptions/sub_abcdef1234567890",
			expected: "/subscriptions/{id}",
		},
		{
			name:         "path with year (should not be treated as ID)",
			trainingURLs: []string{"/archive/2024/posts", "/archive/2024/posts", "/archive/2024/posts"},
			testURL:      "/archive/2024/posts",
			expected:     "/archive/2024/posts",
		},
		{
			name:         "path with small number (should not be treated as ID)",
			trainingURLs: []string{"/page/2", "/page/2", "/page/2"},
			testURL:      "/page/2",
			expected:     "/page/2",
		},
		{
			name: "complex e-commerce URL",
			trainingURLs: []string{
				"/products/electronics/smartphones/iphone-15-pro-987654321/reviews",
				"/products/electronics/smartphones/samsung-s24-ultra-123456789/reviews",
				"/products/electronics/smartphones/pixel-8-pro-555555555/reviews",
			},
			testURL:  "/products/electronics/smartphones/iphone-15-pro-987654321/reviews",
			expected: "/products/electronics/smartphones/{slug}/reviews",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := NewClassifier()
			classifier.Learn(tt.trainingURLs)

			result := classifier.Classify(tt.testURL)
			if result != tt.expected {
				t.Errorf("Classify() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifier_BatchLearning(t *testing.T) {
	tests := []struct {
		name         string
		trainingURLs []string
		testURL      string
		expected     string
	}{
		{
			name: "learn UUID pattern from batch",
			trainingURLs: []string{
				"/projects/d381b052-99eb-40f2-9ede-9bce790faae1/analytics",
				"/projects/a1b2c3d4-e5f6-7890-abcd-ef1234567890/analytics",
				"/projects/12345678-1234-1234-1234-123456789012/analytics",
			},
			testURL:  "/projects/ffffffff-ffff-ffff-ffff-ffffffffffff/analytics",
			expected: "/projects/{uuid}/analytics",
		},
		{
			name: "learn numeric ID pattern from batch",
			trainingURLs: []string{
				"/users/123456/profile",
				"/users/789012/profile",
				"/users/345678/profile",
				"/users/901234/profile",
			},
			testURL:  "/users/555555/profile",
			expected: "/users/{id}/profile",
		},
		{
			name: "distinguish static from dynamic",
			trainingURLs: []string{
				"/api/v1/users/123456/settings",
				"/api/v1/users/789012/settings",
				"/api/v1/users/345678/settings",
			},
			testURL:  "/api/v1/users/999999/settings",
			expected: "/api/v1/users/{id}/settings",
		},
		{
			name: "mixed patterns",
			trainingURLs: []string{
				"/orgs/org-123/projects/d381b052-99eb-40f2-9ede-9bce790faae1/tasks/456789",
				"/orgs/org-456/projects/a1b2c3d4-e5f6-7890-abcd-ef1234567890/tasks/123456",
				"/orgs/org-789/projects/12345678-1234-1234-1234-123456789012/tasks/789012",
			},
			testURL:  "/orgs/org-999/projects/ffffffff-ffff-ffff-ffff-ffffffffffff/tasks/111111",
			expected: "/orgs/{slug}/projects/{uuid}/tasks/{id}",
		},
		{
			name: "keep static segments static",
			trainingURLs: []string{
				"/api/v2/health",
				"/api/v2/health",
				"/api/v2/health",
			},
			testURL:  "/api/v2/health",
			expected: "/api/v2/health",
		},
		{
			name: "dates remain dates",
			trainingURLs: []string{
				"/reports/2024-01-15/summary",
				"/reports/2024-01-16/summary",
				"/reports/2024-01-17/summary",
			},
			testURL:  "/reports/2024-01-18/summary",
			expected: "/reports/{date}/summary",
		},
		{
			name: "hashes remain hashes",
			trainingURLs: []string{
				"/products/507f1f77bcf86cd799439011/details",
				"/products/507f191e810c19729de860ea/details",
				"/products/507f1f77bcf86cd799439999/details",
			},
			testURL:  "/products/507f1f77bcf86cd799439000/details",
			expected: "/products/{hash}/details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := NewClassifier()
			classifier.Learn(tt.trainingURLs)

			result := classifier.Classify(tt.testURL)
			if result != tt.expected {
				t.Errorf("Classify() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifier_Configuration(t *testing.T) {
	t.Run("high cardinality threshold", func(t *testing.T) {
		// With very high threshold, should keep more things static
		classifier := NewClassifier(WithCardinalityThreshold(0.95))

		trainingURLs := []string{
			"/users/123/profile",
			"/users/456/profile",
			"/users/123/profile", // Repeat to lower cardinality
		}
		classifier.Learn(trainingURLs)

		result := classifier.Classify("/users/789/profile")
		// With high threshold, might not parametrize
		t.Logf("Result with high threshold: %s", result)
	})

	t.Run("low cardinality threshold", func(t *testing.T) {
		// With low threshold, should parametrize more aggressively
		classifier := NewClassifier(WithCardinalityThreshold(0.5))

		trainingURLs := []string{
			"/users/123/profile",
			"/users/456/profile",
		}
		classifier.Learn(trainingURLs)

		result := classifier.Classify("/users/789/profile")
		expected := "/users/{id}/profile"
		if result != expected {
			t.Errorf("Classify() = %v, want %v", result, expected)
		}
	})
}

func TestClassifier_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		trainingURLs []string
		testURL      string
		expected     string
	}{
		{
			name:         "empty training set",
			trainingURLs: []string{},
			testURL:      "/users/123/profile",
			expected:     "/users/123/profile", // Without training, keep as-is
		},
		{
			name: "single training URL",
			trainingURLs: []string{
				"/users/123/profile",
			},
			testURL:  "/users/123/profile",
			expected: "/users/123/profile", // Not enough samples for parametrization
		},
		{
			name: "completely new path",
			trainingURLs: []string{
				"/users/123/profile",
				"/users/456/profile",
			},
			testURL:  "/products/abc/details",
			expected: "/products/abc/details", // Unknown path, keep as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classifier := NewClassifier()
			classifier.Learn(tt.trainingURLs)

			result := classifier.Classify(tt.testURL)
			if result != tt.expected {
				t.Errorf("Classify() = %v, want %v", result, tt.expected)
			}
		})
	}
}
