package sw

import (
	"regexp"
	"strings"
)

// ActivityTypeDetector handles detection of activity types from HTML content
type ActivityTypeDetector struct {
	iconPatterns map[string]*regexp.Regexp
	keywords     map[string][]string
}

// NewActivityTypeDetector creates a new activity type detector with predefined patterns
func NewActivityTypeDetector() *ActivityTypeDetector {
	return &ActivityTypeDetector{
		iconPatterns: map[string]*regexp.Regexp{
			"🏃": regexp.MustCompile(`icon.{0,3}running`),
			"🚴": regexp.MustCompile(`regular.biking`),
			"🤸": regexp.MustCompile(`sports-mode`),
		},
		keywords: map[string][]string{
			"🏃":  {"running", "run", "jog", "jogging", "marathon", "5k", "10k", "half marathon"},
			"🚴":  {"cycling", "cycle", "bike", "biking", "bicycle", "mtb", "road bike", "mountain bike"},
			"🏊":  {"swimming", "swim", "pool", "freestyle", "backstroke", "breaststroke", "butterfly"},
			"⛷️": {"skiing", "ski", "alpine", "downhill", "cross country", "nordic", "snowboard", "snowboarding"},
			"🥾":  {"hiking", "hike", "walk", "walking", "trekking", "trail", "nature walk"},
			"💪":  {"gym", "strength", "weight", "lifting", "bodybuilding", "fitness", "workout", "training", "crossfit"},
			"⚽":  {"football", "soccer", "futbol", "match", "league", "pitch"},
			"🏀":  {"basketball", "basket", "court", "dribble", "shoot", "dunk"},
			"🎾":  {"tennis", "court", "racket", "serve", "match", "set"},
			"🚣":  {"rowing", "row", "kayak", "canoe", "paddle", "boat", "crew"},
			"🧘":  {"yoga"},
			"⛳":  {"golf"},
			"🧗":  {"climbing", "boulder"},
			"🛹":  {"skateboard", "skate"},
			"⚾":  {"baseball"},
			"🏐":  {"volleyball"},
		},
	}
}

// DetectActivityType returns the appropriate emoji for an activity type
func (d *ActivityTypeDetector) DetectActivityType(activityType string, fallbackHTML string) string {
	activityType = strings.ToLower(activityType)

	// First, try icon-based detection
	for emoji, pattern := range d.iconPatterns {
		if pattern.MatchString(activityType) {
			return emoji
		}
	}

	// Check for other common activity types in icon class
	if strings.Contains(activityType, "swimming") || strings.Contains(activityType, "swim") {
		return "🏊"
	}
	if strings.Contains(activityType, "hiking") || strings.Contains(activityType, "walk") {
		return "🥾"
	}
	if strings.Contains(activityType, "ski") {
		return "⛷️"
	}
	if strings.Contains(activityType, "gym") || strings.Contains(activityType, "strength") {
		return "💪"
	}

	// If icon-based detection failed, try keyword-based fallback in HTML content
	if fallbackHTML != "" {
		return d.detectFromHTML(fallbackHTML)
	}

	// Default for unknown activity types
	return "❓"
}

// detectFromHTML detects activity type from HTML content using keywords
func (d *ActivityTypeDetector) detectFromHTML(htmlContent string) string {
	content := strings.ToLower(htmlContent)

	// Find the earliest matching keyword by position in the text
	earliestPosition := len(content) + 1
	var matchedEmoji string

	for emoji, keywords := range d.keywords {
		for _, keyword := range keywords {
			keywordPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(keyword) + `\b`)
			if match := keywordPattern.FindStringIndex(content); match != nil {
				if match[0] < earliestPosition {
					earliestPosition = match[0]
					matchedEmoji = emoji
				}
			}
		}
	}

	if matchedEmoji != "" {
		return matchedEmoji
	}

	// Default if no keywords match
	return "❓"
}

// getActivityTypeEmoji returns the appropriate emoji for an activity type (legacy function for backward compatibility)
func getActivityTypeEmoji(activityType string, fallbackHTML string) string {
	detector := NewActivityTypeDetector()
	return detector.DetectActivityType(activityType, fallbackHTML)
}
