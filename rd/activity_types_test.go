package rd

import (
	"testing"
)

func TestActivityTypeDetector_DetectActivityType(t *testing.T) {
	detector := NewActivityTypeDetector()

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
			activityType: "regular-biking",
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
			name:         "swimming in icon class",
			activityType: "icon-swimming",
			fallbackHTML: "",
			expected:     "🏊",
		},
		{
			name:         "hiking in icon class",
			activityType: "icon-hiking",
			fallbackHTML: "",
			expected:     "🥾",
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
			name:         "yoga keyword",
			activityType: "",
			fallbackHTML: `<td>Morning yoga session</td>`,
			expected:     "🧘",
		},
		{
			name:         "golf keyword",
			activityType: "",
			fallbackHTML: `<td>18 holes of golf</td>`,
			expected:     "⛳",
		},
		{
			name:         "climbing keyword",
			activityType: "",
			fallbackHTML: `<td>Rock climbing adventure</td>`,
			expected:     "🧗",
		},
		{
			name:         "unknown activity",
			activityType: "mysterious-sport",
			fallbackHTML: `<td>Some unknown activity</td>`,
			expected:     "❓",
		},
		{
			name:         "case insensitive matching",
			activityType: "ICON-RUNNING",
			fallbackHTML: "",
			expected:     "🏃",
		},
		{
			name:         "partial word matching in HTML",
			activityType: "",
			fallbackHTML: `<td>A nice jog in the neighborhood</td>`,
			expected:     "🏃",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectActivityType(tt.activityType, tt.fallbackHTML)
			if result != tt.expected {
				t.Errorf("DetectActivityType(%q, %q) = %q, want %q",
					tt.activityType, tt.fallbackHTML, result, tt.expected)
			}
		})
	}
}

func TestActivityTypeDetector_DetectFromHTML(t *testing.T) {
	detector := NewActivityTypeDetector()

	tests := []struct {
		name        string
		htmlContent string
		expected    string
	}{
		{
			name:        "running keyword",
			htmlContent: "I went for a run today",
			expected:    "🏃",
		},
		{
			name:        "cycling keyword",
			htmlContent: "Nice cycling session in the morning",
			expected:    "🚴",
		},
		{
			name:        "multiple keywords - first match wins",
			htmlContent: "Running and then some weight training",
			expected:    "🏃", // Should match first found keyword
		},
		{
			name:        "case insensitive",
			htmlContent: "SWIMMING session at the pool",
			expected:    "🏊",
		},
		{
			name:        "word boundary respected",
			htmlContent: "I love programming", // "running" is inside "programming"
			expected:    "❓",                  // Should not match because it's not a word boundary
		},
		{
			name:        "no keywords match",
			htmlContent: "Some random text",
			expected:    "❓",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.detectFromHTML(tt.htmlContent)
			if result != tt.expected {
				t.Errorf("detectFromHTML(%q) = %q, want %q",
					tt.htmlContent, result, tt.expected)
			}
		})
	}
}

func TestGetActivityTypeEmoji_BackwardCompatibility(t *testing.T) {
	// Test the legacy function for backward compatibility
	tests := []struct {
		name         string
		activityType string
		fallbackHTML string
		expected     string
	}{
		{
			name:         "running via legacy function",
			activityType: "icon-running",
			fallbackHTML: "",
			expected:     "🏃",
		},
		{
			name:         "biking via legacy function",
			activityType: "regular-biking",
			fallbackHTML: "",
			expected:     "🚴",
		},
		{
			name:         "unknown via legacy function",
			activityType: "unknown",
			fallbackHTML: "",
			expected:     "❓",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getActivityTypeEmoji(tt.activityType, tt.fallbackHTML)
			if result != tt.expected {
				t.Errorf("getActivityTypeEmoji(%q, %q) = %q, want %q",
					tt.activityType, tt.fallbackHTML, result, tt.expected)
			}
		})
	}
}
