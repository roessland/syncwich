package sw

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ActivityInfo represents information about an activity
type ActivityInfo struct {
	ID         string
	Type       string
	TypeEmoji  string
	Date       string  // Activity date in YYYY-MM-DD format
	DistanceKm float64 // Distance in kilometers
	WeekStart  time.Time
	WeekEnd    time.Time
}

// parseActivitiesFromHTML extracts activity information from HTML content
func parseActivitiesFromHTML(htmlContent []byte, weekStart time.Time, logger interface{}) ([]ActivityInfo, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
	if err != nil {
		return nil, err
	}

	var activities []ActivityInfo
	weekEnd := weekStart.AddDate(0, 0, 6) // End of week (Sunday)

	// Create a detector for type detection
	detector := NewActivityTypeDetector()

	// Track current date for activities that don't show date (same day as previous)
	var currentDate string

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

		// Extract date from health note link (e.g., href="https://runalyze.com/health/note/2025-05-26")
		activityDate := ""
		s.Find("a[href*='/health/note/']").Each(func(j int, a *goquery.Selection) {
			if href, exists := a.Attr("href"); exists {
				// Extract date from URL like ".../health/note/2025-05-26"
				if idx := strings.LastIndex(href, "/"); idx != -1 {
					activityDate = href[idx+1:]
				}
			}
		})

		// If no date found, use the current date (same day as previous activity)
		if activityDate == "" {
			activityDate = currentDate
		} else {
			currentDate = activityDate
		}

		// Extract distance from the row
		distanceKm := parseDistance(s)

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

		// Get emoji using the detector
		emoji := detector.DetectActivityType(activityType, rowHTML)

		// Log unknown activity types for debugging (only if both methods failed)
		if emoji == "❓" && logger != nil {
			// Try to call Debug method via type assertion
			if debugLogger, ok := logger.(interface{ Debug(string, ...any) }); ok {
				debugLogger.Debug("unknown activity type found", "activity_id", activityID, "type", activityType, "row_html_snippet", truncateHTML(rowHTML, 200))
			}
		}

		activities = append(activities, ActivityInfo{
			ID:         activityID,
			Type:       activityType,
			TypeEmoji:  emoji,
			Date:       activityDate,
			DistanceKm: distanceKm,
			WeekStart:  weekStart,
			WeekEnd:    weekEnd,
		})
	})

	return activities, nil
}

// parseDistance extracts distance in km from an activity row
// Handles European format like "6,5 km" or "18,6 km"
func parseDistance(s *goquery.Selection) float64 {
	// Match "6,5 km" but NOT "18,7 km/h" (speed)
	distanceRe := regexp.MustCompile(`(\d+)[,.](\d+)\s*km$`)

	var distance float64
	s.Find("td").Each(func(i int, td *goquery.Selection) {
		// Replace non-breaking space with regular space and trim
		text := strings.TrimSpace(strings.ReplaceAll(td.Text(), "\u00a0", " "))
		if matches := distanceRe.FindStringSubmatch(text); matches != nil {
			// Parse "6,5" as 6.5
			whole := matches[1]
			decimal := matches[2]
			fmt.Sscanf(whole+"."+decimal, "%f", &distance)
		}
	})
	return distance
}

// truncateHTML truncates HTML content for logging
func truncateHTML(html string, maxLen int) string {
	if len(html) <= maxLen {
		return html
	}
	return html[:maxLen] + "..."
}

// FindActivityIds is the regex-based method for extracting activity IDs from HTML
func FindActivityIds(htmlContent []byte) []string {
	re := regexp.MustCompile(`id="training_(\d+)"`)
	matches := re.FindAllStringSubmatch(string(htmlContent), -1)
	ids := make([]string, len(matches))
	for i, match := range matches {
		ids[i] = match[1]
	}
	return ids
}
