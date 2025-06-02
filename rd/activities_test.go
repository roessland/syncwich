package rd

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
				<tr class="r training" id="training_135061340" onclick="Pace.restart();Runalyze.Training.load(135061340, false, event)">
					<td>Some content</td>
				</tr>
			</table>`,
			expected: []string{"135061340"},
		},
		{
			name: "multiple activities",
			html: `<table>
				<tr class="r training" id="training_135061340" onclick="Pace.restart();Runalyze.Training.load(135061340, false, event)">
					<td>First activity</td>
				</tr>
				<tr class="r training" id="training_135061341" onclick="Pace.restart();Runalyze.Training.load(135061341, false, event)">
					<td>Second activity</td>
				</tr>
				<tr class="r training" id="training_135061342" onclick="Pace.restart();Runalyze.Training.load(135061342, false, event)">
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
				<tr class="r training" id="training_135061340" onclick="Pace.restart();Runalyze.Training.load(135061340, false, event)">
					<td>First activity</td>
				</tr>
				<tr class="r other" id="other_456">
					<td>Also not an activity</td>
				</tr>
				<tr class="r training" id="training_135061341" onclick="Pace.restart();Runalyze.Training.load(135061341, false, event)">
					<td>Second activity</td>
				</tr>
			</table>`,
			expected: []string{"135061340", "135061341"},
		},
		{
			name: "malformed HTML",
			html: `<table>
				<tr class="r training" id="training_135061340" onclick="Pace.restart();Runalyze.Training.load(135061340, false, event)">
					<td>First activity
				<tr class="r training" id="training_135061341" onclick="Pace.restart();Runalyze.Training.load(135061341, false, event)">
					<td>Second activity
			</table>`,
			expected: []string{"135061340", "135061341"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findActivityIds([]byte(tt.html))
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
