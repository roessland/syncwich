package rd

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/roessland/runalyzedump/runalyze"
)

// ActivityInfo represents information about an activity
type ActivityInfo struct {
	ID        string
	Type      string
	TypeEmoji string
	WeekStart time.Time
	WeekEnd   time.Time
}

// getActivityTypeEmoji returns the appropriate emoji for an activity type
func getActivityTypeEmoji(activityType string, fallbackHTML string) string {
	activityType = strings.ToLower(activityType)

	// First, try icon-based detection
	runningPattern := regexp.MustCompile(`icon.{0,3}running`)
	bikingPattern := regexp.MustCompile(`regular.biking`)
	sportsPattern := regexp.MustCompile(`sports-mode`)

	if runningPattern.MatchString(activityType) {
		return "🏃"
	}
	if bikingPattern.MatchString(activityType) {
		return "🚴"
	}
	if sportsPattern.MatchString(activityType) {
		return "🤸" // Generic sports person for Runalyze's fallback sports mode
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
		return getActivityTypeFromHTML(fallbackHTML)
	}

	// Default for unknown activity types
	return "❓"
}

// getActivityTypeFromHTML detects activity type from HTML content using keywords
func getActivityTypeFromHTML(htmlContent string) string {
	content := strings.ToLower(htmlContent)

	// Top 10 most common sports with keyword variations
	sportKeywords := map[string]string{
		"🏃":  "running|run|jog|jogging|marathon|5k|10k|half marathon",
		"🚴":  "cycling|cycle|bike|biking|bicycle|mtb|road bike|mountain bike",
		"🏊":  "swimming|swim|pool|freestyle|backstroke|breaststroke|butterfly",
		"⛷️": "skiing|ski|alpine|downhill|cross country|nordic|snowboard|snowboarding",
		"🥾":  "hiking|hike|walk|walking|trekking|trail|nature walk",
		"💪":  "gym|strength|weight|lifting|bodybuilding|fitness|workout|training|crossfit",
		"⚽":  "football|soccer|futbol|match|league|pitch",
		"🏀":  "basketball|basket|court|dribble|shoot|dunk",
		"🎾":  "tennis|court|racket|serve|match|set",
		"🚣":  "rowing|row|kayak|canoe|paddle|boat|crew",
	}

	// Search for keywords in order of specificity
	for emoji, keywords := range sportKeywords {
		keywordPattern := regexp.MustCompile(`\b(` + keywords + `)\b`)
		if keywordPattern.MatchString(content) {
			return emoji
		}
	}

	// Additional common activities
	if strings.Contains(content, "yoga") {
		return "🧘"
	}
	if strings.Contains(content, "golf") {
		return "⛳"
	}
	if strings.Contains(content, "climbing") || strings.Contains(content, "boulder") {
		return "🧗"
	}
	if strings.Contains(content, "skateboard") || strings.Contains(content, "skate") {
		return "🛹"
	}
	if strings.Contains(content, "baseball") {
		return "⚾"
	}
	if strings.Contains(content, "volleyball") {
		return "🏐"
	}

	// Default if no keywords match
	return "❓"
}

// parseActivitiesFromHTML extracts activity information from HTML content
func parseActivitiesFromHTML(htmlContent []byte, weekStart time.Time, logger interface{}) ([]ActivityInfo, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
	if err != nil {
		return nil, err
	}

	var activities []ActivityInfo
	weekEnd := weekStart.AddDate(0, 0, 6) // End of week (Sunday)

	// Find all training rows
	doc.Find("tr[id^='training_']").Each(func(i int, s *goquery.Selection) {
		// Extract activity ID from the id attribute
		id, exists := s.Attr("id")
		if !exists {
			return
		}

		// Remove "training_" prefix to get just the ID
		activityID := strings.TrimPrefix(id, "training_")

		// Get the entire row HTML for fallback detection
		rowHTML, _ := s.Html()

		// Find the activity type icon/class
		activityType := ""
		s.Find("td").First().Find("i").Each(func(j int, iconSel *goquery.Selection) {
			if class, exists := iconSel.Attr("class"); exists {
				activityType = class
				return
			}
		})

		// If no icon found, try to find it in other ways
		if activityType == "" {
			s.Find("i[class*='icon']").Each(func(j int, iconSel *goquery.Selection) {
				if class, exists := iconSel.Attr("class"); exists {
					activityType = class
					return
				}
			})
		}

		// Get emoji using both icon class and HTML fallback
		emoji := getActivityTypeEmoji(activityType, rowHTML)

		// Log unknown activity types for debugging (only if both methods failed)
		if emoji == "❓" && logger != nil {
			// Try to call Debug method via type assertion
			if debugLogger, ok := logger.(interface{ Debug(string, ...any) }); ok {
				debugLogger.Debug("unknown activity type found", "activity_id", activityID, "type", activityType, "row_html_snippet", truncateHTML(rowHTML, 200))
			}
		}

		activities = append(activities, ActivityInfo{
			ID:        activityID,
			Type:      activityType,
			TypeEmoji: emoji,
			WeekStart: weekStart,
			WeekEnd:   weekEnd,
		})
	})

	return activities, nil
}

// truncateHTML truncates HTML content for logging
func truncateHTML(html string, maxLen int) string {
	if len(html) <= maxLen {
		return html
	}
	return html[:maxLen] + "..."
}

// ActivityIterator is an iterator that yields activity information from Runalyze
type ActivityIterator struct {
	client        *runalyze.Client
	ctx           context.Context
	untilDate     time.Time
	sinceDate     time.Time
	done          bool
	activities    []ActivityInfo
	activityIndex int
	logger        interface{} // We'll accept any logger interface
}

// NewActivityIterator creates a new ActivityIterator starting from the given date
func NewActivityIterator(client *runalyze.Client, untilDate time.Time) *ActivityIterator {
	return &ActivityIterator{
		client:        client,
		ctx:           context.Background(),
		untilDate:     untilDate,
		sinceDate:     time.Time{}, // Zero time means no limit
		done:          false,
		activities:    nil,
		activityIndex: 0,
	}
}

// NewActivityIteratorWithSince creates a new ActivityIterator starting from the given date and stopping at the since date
func NewActivityIteratorWithSince(client *runalyze.Client, untilDate, sinceDate time.Time) *ActivityIterator {
	return &ActivityIterator{
		client:        client,
		ctx:           context.Background(),
		untilDate:     untilDate,
		sinceDate:     sinceDate,
		done:          false,
		activities:    nil,
		activityIndex: 0,
	}
}

// SetLogger sets a logger for the iterator (optional)
func (it *ActivityIterator) SetLogger(logger interface{}) {
	it.logger = logger
}

// fetchActivitiesForWeek fetches activities for the current week
func (it *ActivityIterator) fetchActivitiesForWeek() error {
	// Check if we've gone beyond the since date
	if !it.sinceDate.IsZero() && it.untilDate.Before(it.sinceDate) {
		it.done = true
		return nil
	}

	// Get the data browser page for this week
	data, err := it.client.GetDataBrowser(it.untilDate)
	if err != nil {
		return err
	}

	// Parse activities from HTML
	activities, err := parseActivitiesFromHTML(data, it.untilDate, it.logger)
	if err != nil {
		// Log error but don't fail completely, fall back to regex
		if it.logger != nil {
			// Use reflection or type assertion to call Debug method if available
			// For now, we'll just ignore the error and fall back
		}

		// Fallback to old regex method
		ids := findActivityIdsRegex(data)
		activities = make([]ActivityInfo, len(ids))
		for i, id := range ids {
			activities[i] = ActivityInfo{
				ID:        id,
				Type:      "unknown",
				TypeEmoji: getActivityTypeEmoji("unknown", ""), // Pass empty string for HTML fallback
				WeekStart: it.untilDate,
				WeekEnd:   it.untilDate.AddDate(0, 0, 6),
			}
		}
	}

	it.activities = activities
	it.activityIndex = 0

	// Move to the previous week
	it.untilDate = it.untilDate.AddDate(0, 0, -7)

	return nil
}

// findActivityIdsRegex is the old regex-based fallback method
func findActivityIdsRegex(htmlContent []byte) []string {
	re := regexp.MustCompile(`id="training_(\d+)"`)
	matches := re.FindAllStringSubmatch(string(htmlContent), -1)
	ids := make([]string, len(matches))
	for i, match := range matches {
		ids[i] = match[1]
	}
	return ids
}

// Next returns the next activity info and whether there are more activities
func (it *ActivityIterator) Next() (ActivityInfo, bool) {
	if it.done {
		return ActivityInfo{}, false
	}

	// If we've consumed all activities in the current week, fetch the next week
	if it.activityIndex >= len(it.activities) {
		err := it.fetchActivitiesForWeek()
		if err != nil {
			it.done = true
			return ActivityInfo{}, false
		}

		// If we got no activities, try the next week
		if len(it.activities) == 0 {
			return it.Next()
		}
	}

	// Double-check bounds after potential recursive call
	if it.activityIndex >= len(it.activities) {
		it.done = true
		return ActivityInfo{}, false
	}

	// Return the next activity info
	activity := it.activities[it.activityIndex]
	it.activityIndex++
	return activity, true
}

// Example usage:
// iter := NewActivityIterator(client, time.Now())
// for activity, ok := iter.Next(); ok; activity, ok = iter.Next() {
//     fmt.Println(activity)
// }
