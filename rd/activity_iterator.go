package rd

import (
	"context"
	"time"

	"github.com/roessland/runalyzedump/runalyze"
)

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
		ids := FindActivityIds(data)
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
