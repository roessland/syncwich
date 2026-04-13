package sw

import (
	"testing"
	"time"
)

func TestValidateAndParseDates_SpecificDates(t *testing.T) {
	tests := []struct {
		name      string
		untilStr  string
		sinceStr  string
		wantSince string
		wantUntil string
	}{
		{
			name:      "specific until and since dates",
			untilStr:  "2024-01-15",
			sinceStr:  "2024-01-01",
			wantSince: "2024-01-01",
			wantUntil: "2024-01-15", // Should stay Monday (it's a Monday)
		},
		{
			name:      "month-only until date",
			untilStr:  "2024-01",
			sinceStr:  "2023-12-01",
			wantSince: "2023-12-04", // 2023-12-01 is Friday, transforms to next Monday
			wantUntil: "2024-02-05", // Next Monday after Jan 31
		},
		{
			name:      "year-only until date",
			untilStr:  "2024",
			sinceStr:  "2024-01-01",
			wantSince: "2024-01-01",
			wantUntil: "2025-01-06", // Next Monday after Dec 31, 2024
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			since, until, err := ValidateAndParseDates(tt.untilStr, tt.sinceStr)
			if err != nil {
				t.Errorf("ValidateAndParseDates() error = %v", err)
				return
			}

			if since.Format("2006-01-02") != tt.wantSince {
				t.Errorf("since = %v, want %v", since.Format("2006-01-02"), tt.wantSince)
			}
			if until.Format("2006-01-02") != tt.wantUntil {
				t.Errorf("until = %v, want %v", until.Format("2006-01-02"), tt.wantUntil)
			}
		})
	}
}

func TestValidateAndParseDates_DurationSince(t *testing.T) {
	tests := []struct {
		name            string
		untilStr        string
		sinceStr        string
		expectedDayDiff int
	}{
		{
			name:            "30 days duration",
			untilStr:        "2024-01-29", // Monday
			sinceStr:        "30d",
			expectedDayDiff: 30,
		},
		{
			name:            "2 weeks duration",
			untilStr:        "2024-01-29", // Monday
			sinceStr:        "2w",
			expectedDayDiff: 14,
		},
		{
			name:            "6 months duration (approximate)",
			untilStr:        "2024-07-01", // Monday
			sinceStr:        "6m",
			expectedDayDiff: 180, // 6 * 30 days
		},
		{
			name:            "1 year duration (approximate)",
			untilStr:        "2024-01-01", // Monday
			sinceStr:        "1y",
			expectedDayDiff: 365,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			since, until, err := ValidateAndParseDates(tt.untilStr, tt.sinceStr)
			if err != nil {
				t.Errorf("ValidateAndParseDates() error = %v", err)
				return
			}

			actualDiff := calendarDaysBetween(since, until)
			if actualDiff != tt.expectedDayDiff {
				t.Errorf("day difference = %d, want %d", actualDiff, tt.expectedDayDiff)
			}
		})
	}
}

func TestValidateAndParseDates_Defaults(t *testing.T) {
	// Test with empty strings - should use sensible defaults
	since, until, err := ValidateAndParseDates("", "")
	if err != nil {
		t.Errorf("ValidateAndParseDates() with empty strings error = %v", err)
		return
	}

	// Until should be next Monday from now
	now := time.Now()
	daysUntilMonday := (8 - int(now.Weekday())) % 7
	expectedUntil := now.AddDate(0, 0, daysUntilMonday)
	expectedUntil = time.Date(expectedUntil.Year(), expectedUntil.Month(), expectedUntil.Day(), 0, 0, 0, 0, expectedUntil.Location())

	if until.Format("2006-01-02") != expectedUntil.Format("2006-01-02") {
		t.Errorf("until = %v, want %v", until.Format("2006-01-02"), expectedUntil.Format("2006-01-02"))
	}

	// Since should be 4 weeks before until
	expectedSince := until.AddDate(0, 0, -28) // 4 weeks = 28 days
	if since.Format("2006-01-02") != expectedSince.Format("2006-01-02") {
		t.Errorf("since = %v, want %v", since.Format("2006-01-02"), expectedSince.Format("2006-01-02"))
	}

	// Verify it's actually 28 calendar days difference. Must count by
	// calendar days, not wall-clock hours, because DST transitions make
	// 28 days != 28*24 hours in DST-observing zones.
	if dayDiff := calendarDaysBetween(since, until); dayDiff != 28 {
		t.Errorf("day difference = %d, want 28", dayDiff)
	}
}

// calendarDaysBetween returns the number of calendar days between two times,
// ignoring DST-driven hour drift by anchoring both at UTC midnight.
func calendarDaysBetween(from, to time.Time) int {
	fromMidnight := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toMidnight := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(toMidnight.Sub(fromMidnight).Hours() / 24)
}

// TestParseSinceDate_DSTBoundary locks in DST-safe behaviour for day and week
// durations. Before the fix, `since = until.Add(-n*24h)` silently lost or
// gained an hour across a DST transition, which formatted as the wrong date.
func TestParseSinceDate_DSTBoundary(t *testing.T) {
	oslo, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Skipf("Europe/Oslo tzdata not available: %v", err)
	}

	// Europe/Oslo DST starts on 2026-03-29. Pick an until date after DST,
	// subtract a span that crosses the transition.
	until := time.Date(2026, 4, 20, 0, 0, 0, 0, oslo)

	tests := []struct {
		name  string
		since string
		want  string // Expected since date in YYYY-MM-DD (local).
	}{
		{"4 weeks crossing DST start", "4w", "2026-03-23"},
		{"28 days crossing DST start", "28d", "2026-03-23"},
		{"1 week, same side of DST", "1w", "2026-04-13"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			since, err := parseSinceDate(tt.since, until)
			if err != nil {
				t.Fatalf("parseSinceDate(%q, %v): %v", tt.since, until, err)
			}
			if got := since.Format("2006-01-02"); got != tt.want {
				t.Errorf("parseSinceDate(%q, %v) = %s, want %s", tt.since, until.Format("2006-01-02"), got, tt.want)
			}
		})
	}
}

// TestValidateAndParseDates_DefaultsDSTStable verifies the implicit 4-week
// default window is DST-safe by forcing the ambient clock to a post-DST
// moment via TZ-aware arithmetic against a known until.
func TestValidateAndParseDates_DefaultsDSTStable(t *testing.T) {
	oslo, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		t.Skipf("Europe/Oslo tzdata not available: %v", err)
	}

	// 28 calendar days before a post-DST Monday is a pre-DST Monday; the
	// difference must be exactly 28 days regardless of wall-clock drift.
	until := time.Date(2026, 4, 20, 0, 0, 0, 0, oslo)
	since, err := parseSinceDate("4w", until)
	if err != nil {
		t.Fatalf("parseSinceDate: %v", err)
	}

	gotUntilDate := until.Format("2006-01-02")
	gotSinceDate := since.Format("2006-01-02")
	if gotSinceDate != "2026-03-23" {
		t.Errorf("since date = %s, want 2026-03-23 (until=%s)", gotSinceDate, gotUntilDate)
	}
	// Midnight in local tz — not some DST-shifted minute.
	if h, m, s := since.Clock(); h != 0 || m != 0 || s != 0 {
		t.Errorf("since clock = %02d:%02d:%02d, want 00:00:00", h, m, s)
	}
}

func TestValidateAndParseDates_ValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		untilStr string
		sinceStr string
		wantErr  bool
	}{
		{
			name:     "invalid until date format",
			untilStr: "2024-13-45", // Invalid date
			sinceStr: "2024-01-01",
			wantErr:  true,
		},
		{
			name:     "invalid since date format",
			untilStr: "2024-01-15",
			sinceStr: "2024-25-99", // Invalid date
			wantErr:  true,
		},
		{
			name:     "invalid duration format",
			untilStr: "2024-01-15",
			sinceStr: "30x", // Invalid duration unit
			wantErr:  true,
		},
		{
			name:     "since after until",
			untilStr: "2024-01-01",
			sinceStr: "2024-01-15", // After until
			wantErr:  true,
		},
		{
			name:     "since equals until",
			untilStr: "2024-01-01",
			sinceStr: "2024-01-01", // Same as until
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ValidateAndParseDates(tt.untilStr, tt.sinceStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndParseDates() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAndParseDates_MondayTransformation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // Expected Monday
	}{
		{
			name:     "Tuesday becomes next Monday",
			input:    "2024-01-02", // Tuesday
			expected: "2024-01-08", // Next Monday
		},
		{
			name:     "Sunday becomes next Monday",
			input:    "2024-01-07", // Sunday
			expected: "2024-01-08", // Next Monday
		},
		{
			name:     "Monday stays Monday",
			input:    "2024-01-01", // Monday
			expected: "2024-01-01", // Same Monday
		},
		{
			name:     "Friday becomes next Monday",
			input:    "2024-01-05", // Friday
			expected: "2024-01-08", // Next Monday
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, until, err := ValidateAndParseDates(tt.input, "2023-01-01") // Use fixed old since date
			if err != nil {
				t.Errorf("ValidateAndParseDates() error = %v", err)
				return
			}

			if until.Format("2006-01-02") != tt.expected {
				t.Errorf("until = %v, want %v", until.Format("2006-01-02"), tt.expected)
			}

			// Verify it's actually a Monday
			if until.Weekday() != time.Monday {
				t.Errorf("until is %v, want Monday", until.Weekday())
			}
		})
	}
}
