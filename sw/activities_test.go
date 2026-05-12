package sw

import (
	"testing"
)

func TestFindActivityIds(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected []string
	}{
		{
			name: "single activity",
			html: `<table>
				<tr class="text-right" data-load-target="#activity" data-load-url="https://runalyze.com/activity/135061340" data-activity-id="135061340">
					<td>Some content</td>
				</tr>
			</table>`,
			expected: []string{"135061340"},
		},
		{
			name: "multiple activities",
			html: `<table>
				<tr class="text-right" data-load-target="#activity" data-load-url="https://runalyze.com/activity/135061340" data-activity-id="135061340">
					<td>First activity</td>
				</tr>
				<tr class="text-right" data-load-target="#activity" data-load-url="https://runalyze.com/activity/135061341" data-activity-id="135061341">
					<td>Second activity</td>
				</tr>
				<tr class="text-right" data-load-target="#activity" data-load-url="https://runalyze.com/activity/135061342" data-activity-id="135061342">
					<td>Third activity</td>
				</tr>
			</table>`,
			expected: []string{"135061340", "135061341", "135061342"},
		},
		{
			name: "no activities",
			html: `<table>
				<tr class="r other" id="other_123">
					<td>Not an activity</td>
				</tr>
			</table>`,
			expected: []string{},
		},
		{
			name: "mixed content",
			html: `<table>
				<tr class="r other" id="other_123">
					<td>Not an activity</td>
				</tr>
				<tr class="text-right" data-load-target="#activity" data-load-url="https://runalyze.com/activity/135061340" data-activity-id="135061340">
					<td>First activity</td>
				</tr>
				<tr class="r other" id="other_456">
					<td>Also not an activity</td>
				</tr>
				<tr class="text-right" data-load-target="#activity" data-load-url="https://runalyze.com/activity/135061341" data-activity-id="135061341">
					<td>Second activity</td>
				</tr>
			</table>`,
			expected: []string{"135061340", "135061341"},
		},
		{
			name: "malformed HTML",
			html: `<table>
				<tr class="text-right" data-load-target="#activity" data-load-url="https://runalyze.com/activity/135061340" data-activity-id="135061340">
					<td>First activity
				<tr class="text-right" data-load-target="#activity" data-load-url="https://runalyze.com/activity/135061341" data-activity-id="135061341">
					<td>Second activity
			</table>`,
			expected: []string{"135061340", "135061341"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindActivityIds([]byte(tt.html))
			if len(got) != len(tt.expected) {
				t.Errorf("findActivityIds() returned %d IDs, want %d", len(got), len(tt.expected))
				return
			}
			for i, id := range got {
				if id != tt.expected[i] {
					t.Errorf("findActivityIds()[%d] = %s, want %s", i, id, tt.expected[i])
				}
			}
		})
	}
}
