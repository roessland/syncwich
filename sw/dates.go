package sw

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// parseUntilDate parses a date string in YYYY-MM-DD, YYYY-MM, or YYYY format and returns the next Monday
func parseUntilDate(dateStr string) (time.Time, error) {
	var t time.Time
	var err error

	// Try parsing as YYYY-MM-DD
	t, err = time.Parse("2006-01-02", dateStr)
	if err != nil {
		// Try parsing as YYYY-MM
		t, err = time.Parse("2006-01", dateStr)
		if err != nil {
			// Try parsing as YYYY
			t, err = time.Parse("2006", dateStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid date format. Use YYYY-MM-DD, YYYY-MM, or YYYY")
			}
			// If it's just a year, use the last day of the year (December 31st)
			t = time.Date(t.Year(), 12, 31, 0, 0, 0, 0, t.Location())
		} else {
			// If it's just a month, use the last day of the month
			t = time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location())
		}
	}

	// If it's already a Monday, return it
	if t.Weekday() == time.Monday {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
	}

	// Find the next Monday
	daysUntilMonday := (8 - int(t.Weekday())) % 7
	nextMonday := t.AddDate(0, 0, daysUntilMonday)
	return time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 0, 0, 0, 0, nextMonday.Location()), nil
}

// parseDuration parses a simplified prometheus-style duration string
// Supports: y (years), w (weeks), d (days), m (months)
// Examples: "30d", "2w", "1y", "6m"
// No combinations allowed (e.g., "1y2w" is invalid)
func parseDuration(durationStr string) (time.Duration, error) {
	// Regex to match the simplified duration format
	re := regexp.MustCompile(`^([0-9]+)([ywdm])$`)
	matches := re.FindStringSubmatch(durationStr)

	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid duration format. Use format like '30d', '2w', '1y', or '6m' (no combinations allowed)")
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", matches[1])
	}

	unit := matches[2]

	switch unit {
	case "y":
		// Approximate: 365 days per year
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "m":
		// Approximate: 30 days per month
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid duration unit: %s (use y, w, d, or m)", unit)
	}
}

// parseSinceDate parses a --since parameter which can be either:
// - A date string (YYYY-MM-DD, YYYY-MM, or YYYY format)
// - A duration string (30d, 2w, 1y, 6m) - relative to the until date
func parseSinceDate(sinceStr string, untilDate time.Time) (time.Time, error) {
	// First, try to parse as a duration
	if duration, err := parseDuration(sinceStr); err == nil {
		// Calculate the date by subtracting the duration from the until date
		return untilDate.Add(-duration), nil
	}

	// If not a duration, try to parse as a date using the existing logic
	return parseUntilDate(sinceStr)
}

// ValidateAndParseDates validates and parses the until and since date parameters early
func ValidateAndParseDates(untilStr, sinceStr string) (since, until time.Time, err error) {
	// Parse until date
	if untilStr != "" {
		until, err = parseUntilDate(untilStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse until date: %w", err)
		}
	} else {
		// Default to current time, transformed to next Monday
		until = time.Now()
		daysUntilMonday := (8 - int(until.Weekday())) % 7
		until = until.AddDate(0, 0, daysUntilMonday)
		until = time.Date(until.Year(), until.Month(), until.Day(), 0, 0, 0, 0, until.Location())
	}

	// Parse since date (relative to until date)
	if sinceStr != "" {
		since, err = parseSinceDate(sinceStr, until)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse since date: %w", err)
		}
	} else {
		// Default to 4 weeks before the until date
		defaultDuration, _ := parseDuration("4w")
		since = until.Add(-defaultDuration)
	}

	// Validate that since is before until
	if since.After(until) || since.Equal(until) {
		return time.Time{}, time.Time{}, fmt.Errorf("--since date (%s) must be before --until date (%s)", since.Format("2006-01-02"), until.Format("2006-01-02"))
	}

	return since, until, nil
}
