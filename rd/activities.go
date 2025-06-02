package rd

import (
	"context"
	"regexp"
	"time"

	"github.com/roessland/runalyzedump/runalyze"
)

// findActivityIds extracts activity IDs from HTML content
func findActivityIds(htmlContent []byte) []string {
	// Match the training ID from the id attribute
	re := regexp.MustCompile(`id="training_(\d+)"`)
	matches := re.FindAllStringSubmatch(string(htmlContent), -1)
	ids := make([]string, len(matches))
	for i, match := range matches {
		ids[i] = match[1]
	}
	return ids
}

// ActivityIterator is an iterator that yields activity IDs from Runalyze
type ActivityIterator struct {
	client        *runalyze.Client
	ctx           context.Context
	startDate     time.Time
	done          bool
	activities    []string
	activityIndex int
}

// NewActivityIterator creates a new ActivityIterator starting from the given date
func NewActivityIterator(client *runalyze.Client, startDate time.Time) *ActivityIterator {
	return &ActivityIterator{
		client:        client,
		ctx:           context.Background(),
		startDate:     startDate,
		done:          false,
		activities:    nil,
		activityIndex: 0,
	}
}

// fetchActivitiesForWeek fetches activities for the current week
func (it *ActivityIterator) fetchActivitiesForWeek() error {
	// Get the data browser page for this week
	data, err := it.client.GetDataBrowser(it.startDate)
	if err != nil {
		return err
	}

	// Extract activity IDs
	it.activities = findActivityIds(data)
	it.activityIndex = 0

	// Move to the previous week
	it.startDate = it.startDate.AddDate(0, 0, -7)

	return nil
}

// Next returns the next activity ID and whether there are more activities
func (it *ActivityIterator) Next() (string, bool) {
	if it.done {
		return "", false
	}

	// If we've consumed all activities in the current week, fetch the next week
	if it.activityIndex >= len(it.activities) {
		err := it.fetchActivitiesForWeek()
		if err != nil {
			it.done = true
			return "", false
		}

		// If we got no activities, try the next week
		if len(it.activities) == 0 {
			return it.Next()
		}
	}

	// Return the next activity ID
	activityID := it.activities[it.activityIndex]
	it.activityIndex++
	return activityID, true
}

// Example usage:
// iter := NewActivityIterator(client, time.Now())
// for activityID, ok := iter.Next(); ok; activityID, ok = iter.Next() {
//     fmt.Println(activityID)
// }
