package sw

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var updateGolden = flag.Bool("update-golden", false, "Update golden master files")

// TestParseActivitiesFromHTML_Fixtures tests parsing with real Runalyze HTML fixtures
func TestParseActivitiesFromHTML_Fixtures(t *testing.T) {
	fixturesDir := filepath.Join("testdata", "fixtures")
	goldenDir := filepath.Join("testdata", "golden")

	fixtures, err := filepath.Glob(filepath.Join(fixturesDir, "*.html"))
	if err != nil {
		t.Fatalf("Failed to find fixture files: %v", err)
	}

	if len(fixtures) == 0 {
		t.Skip("No fixture files found - run 'just update-fixtures' to generate them")
	}

	for _, fixturePath := range fixtures {
		fixtureName := filepath.Base(fixturePath)
		testName := fixtureName[:len(fixtureName)-5] // Remove .html extension

		t.Run(testName, func(t *testing.T) {
			// Read HTML fixture
			htmlData, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("Failed to read fixture %s: %v", fixturePath, err)
			}

			// Parse week start from filename (e.g., "2024.01.01-week.html")
			weekStart, err := time.Parse("2006.01.02", testName[:10])
			if err != nil {
				t.Fatalf("Failed to parse week start from %s: %v", testName, err)
			}

			// Parse activities
			mockLogger := &MockLogger{}
			activities, err := parseActivitiesFromHTML(htmlData, weekStart, mockLogger)
			if err != nil {
				t.Fatalf("parseActivitiesFromHTML failed: %v", err)
			}

			// Verify we have some activities (fixture should have 2+)
			if len(activities) < 2 {
				t.Logf("Warning: Only %d activities found (expected 2+)", len(activities))
			}

			// Convert to JSON for comparison
			actualJSON, err := json.MarshalIndent(activities, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal activities: %v", err)
			}

			goldenPath := filepath.Join(goldenDir, testName+".json")

			if *updateGolden {
				// Update golden file
				if err := os.MkdirAll(goldenDir, 0755); err != nil {
					t.Fatalf("Failed to create golden dir: %v", err)
				}
				if err := os.WriteFile(goldenPath, actualJSON, 0644); err != nil {
					t.Fatalf("Failed to write golden file: %v", err)
				}
				t.Logf("Updated golden file: %s", goldenPath)
				return
			}

			// Compare with golden file
			expectedJSON, err := os.ReadFile(goldenPath)
			if err != nil {
				if os.IsNotExist(err) {
					t.Fatalf("Golden file missing: %s\n\nRun 'just update-golden' to create it.\n\nActual result:\n%s", goldenPath, string(actualJSON))
				}
				t.Fatalf("Failed to read golden file: %v", err)
			}

			if string(actualJSON) != string(expectedJSON) {
				t.Errorf("Activities parsing mismatch for %s\n\nExpected:\n%s\n\nActual:\n%s\n\nTo update: just update-golden", testName, string(expectedJSON), string(actualJSON))
			}

			t.Logf("✅ Parsed %d activities successfully", len(activities))
		})
	}
}

// TestActivityTypeEmoji_EdgeCases tests emoji detection with various HTML patterns
func TestActivityTypeEmoji_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		activityType string
		fallbackHTML string
		expected     string
	}{
		{
			name:         "running icon class",
			activityType: "icon-running",
			fallbackHTML: "",
			expected:     "🏃",
		},
		{
			name:         "biking icon class",
			activityType: "regular biking",
			fallbackHTML: "",
			expected:     "🚴",
		},
		{
			name:         "sports mode fallback",
			activityType: "sports-mode",
			fallbackHTML: "",
			expected:     "🤸",
		},
		{
			name:         "fallback to HTML content - running",
			activityType: "unknown",
			fallbackHTML: `<td>Morning run in the park</td>`,
			expected:     "🏃",
		},
		{
			name:         "fallback to HTML content - cycling",
			activityType: "unknown",
			fallbackHTML: `<td>Evening bike ride through the mountains</td>`,
			expected:     "🚴",
		},
		{
			name:         "swimming in HTML",
			activityType: "",
			fallbackHTML: `<td>Pool swimming session</td>`,
			expected:     "🏊",
		},
		{
			name:         "hiking keywords",
			activityType: "",
			fallbackHTML: `<td>Mountain hiking trail</td>`,
			expected:     "🥾",
		},
		{
			name:         "gym workout",
			activityType: "",
			fallbackHTML: `<td>Strength training at the gym</td>`,
			expected:     "💪",
		},
		{
			name:         "unknown activity",
			activityType: "mysterious-sport",
			fallbackHTML: `<td>Some unknown activity</td>`,
			expected:     "❓",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getActivityTypeEmoji(tt.activityType, tt.fallbackHTML)
			if result != tt.expected {
				t.Errorf("getActivityTypeEmoji(%q, %q) = %q, want %q", tt.activityType, tt.fallbackHTML, result, tt.expected)
			}
		})
	}
}

// TestParseActivitiesFromHTML_ExtractsData verifies we extract meaningful data from activities
func TestParseActivitiesFromHTML_ExtractsData(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixtures", "2025.05.26-week.html")

	htmlData, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Skip("Fixture not found - run 'just update-fixtures' first")
	}

	weekStart, _ := time.Parse("2006.01.02", "2025.05.26")
	activities, err := parseActivitiesFromHTML(htmlData, weekStart, nil)
	if err != nil {
		t.Fatalf("parseActivitiesFromHTML failed: %v", err)
	}

	// Should have 6 activities for this week
	if len(activities) != 6 {
		t.Fatalf("Expected 6 activities, got %d", len(activities))
	}

	// First activity: Monday run, 6.5 km
	first := activities[0]
	if first.ID != "135061341" {
		t.Errorf("First activity ID = %q, want %q", first.ID, "135061341")
	}
	if first.Date != "2025-05-26" {
		t.Errorf("First activity Date = %q, want %q", first.Date, "2025-05-26")
	}
	if first.DistanceKm < 6.4 || first.DistanceKm > 6.6 {
		t.Errorf("First activity DistanceKm = %v, want ~6.5", first.DistanceKm)
	}
	if first.TypeEmoji != "🏃" {
		t.Errorf("First activity TypeEmoji = %q, want 🏃", first.TypeEmoji)
	}

	// Second activity: Monday cycling, 2.6 km (same day, no date in HTML)
	second := activities[1]
	if second.ID != "135061340" {
		t.Errorf("Second activity ID = %q, want %q", second.ID, "135061340")
	}
	if second.Date != "2025-05-26" {
		t.Errorf("Second activity Date = %q, want %q (should inherit from previous)", second.Date, "2025-05-26")
	}
	if second.DistanceKm < 2.5 || second.DistanceKm > 2.7 {
		t.Errorf("Second activity DistanceKm = %v, want ~2.6", second.DistanceKm)
	}
	if second.TypeEmoji != "🚴" {
		t.Errorf("Second activity TypeEmoji = %q, want 🚴", second.TypeEmoji)
	}
}

// TestFindActivityIds_WithFixtures tests the regex-based ID extraction with fixture data
func TestFindActivityIds_WithFixtures(t *testing.T) {
	fixturesDir := filepath.Join("testdata", "fixtures")

	// Check if fixtures exist
	fixtures, err := filepath.Glob(filepath.Join(fixturesDir, "*.html"))
	if err != nil || len(fixtures) == 0 {
		t.Skip("No fixture files found - run 'just update-fixtures' to generate them")
	}

	for _, fixturePath := range fixtures {
		fixtureName := filepath.Base(fixturePath)
		testName := fixtureName[:len(fixtureName)-5] // Remove .html extension

		t.Run(testName, func(t *testing.T) {
			// Read HTML fixture
			htmlData, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("Failed to read fixture %s: %v", fixturePath, err)
			}

			// Extract activity IDs using regex method
			ids := FindActivityIds(htmlData)

			// Verify we have reasonable results (fixture should have 2+)
			if len(ids) < 2 {
				t.Logf("Warning: Only %d activity IDs found (expected 2+)", len(ids))
			}

			// Verify all IDs are numeric and non-empty
			for i, id := range ids {
				if len(id) == 0 {
					t.Errorf("Activity ID %d is empty", i)
				}
				// Check if it's all digits (basic sanity check)
				for _, r := range id {
					if r < '0' || r > '9' {
						t.Errorf("Activity ID %s contains non-digit character: %c", id, r)
					}
				}
			}

			t.Logf("✅ Found %d activity IDs", len(ids))
		})
	}
}
