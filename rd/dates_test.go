package rd

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

			actualDiff := int(until.Sub(since).Hours() / 24)
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

	// Verify it's actually 28 days difference
	dayDiff := int(until.Sub(since).Hours() / 24)
	if dayDiff != 28 {
		t.Errorf("day difference = %d, want 28", dayDiff)
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
