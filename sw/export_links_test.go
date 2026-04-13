package sw

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/roessland/syncwich/runalyze"
)

// TestExtractExportLinks_Fixtures parses real Runalyze activity pages and compares
// the extracted export links against golden files. Guards against Runalyze changing
// the export URL scheme (e.g. `/export/file/fit` vs `/export/file/fit-original`).
func TestExtractExportLinks_Fixtures(t *testing.T) {
	fixturesDir := filepath.Join("testdata", "fixtures")
	goldenDir := filepath.Join("testdata", "golden")

	fixtures, err := filepath.Glob(filepath.Join(fixturesDir, "activity-*.html"))
	if err != nil {
		t.Fatalf("Failed to find activity fixture files: %v", err)
	}

	if len(fixtures) == 0 {
		t.Skip("No activity fixture files found - run 'just update-fixtures' to generate them")
	}

	for _, fixturePath := range fixtures {
		fixtureName := filepath.Base(fixturePath)
		testName := strings.TrimSuffix(fixtureName, ".html")

		t.Run(testName, func(t *testing.T) {
			htmlData, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("Failed to read fixture %s: %v", fixturePath, err)
			}

			links := ExtractExportLinks(htmlData)

			actualJSON, err := json.MarshalIndent(links, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal links: %v", err)
			}

			goldenPath := filepath.Join(goldenDir, testName+".json")

			if *updateGolden {
				if err := os.MkdirAll(goldenDir, 0755); err != nil {
					t.Fatalf("Failed to create golden dir: %v", err)
				}
				if err := os.WriteFile(goldenPath, actualJSON, 0644); err != nil {
					t.Fatalf("Failed to write golden file: %v", err)
				}
				t.Logf("Updated golden file: %s", goldenPath)
				return
			}

			expectedJSON, err := os.ReadFile(goldenPath)
			if err != nil {
				if os.IsNotExist(err) {
					t.Fatalf("Golden file missing: %s\n\nRun 'just update-golden' to create it.\n\nActual result:\n%s", goldenPath, string(actualJSON))
				}
				t.Fatalf("Failed to read golden file: %v", err)
			}

			if string(actualJSON) != string(expectedJSON) {
				t.Errorf("Export links mismatch for %s\n\nExpected:\n%s\n\nActual:\n%s\n\nTo update: just update-golden", testName, string(expectedJSON), string(actualJSON))
			}

			// Invariant: every activity page must advertise FIT (original) and TCX exports.
			// These are the two formats syncwich downloads — if Runalyze removes them,
			// downloads silently 403.
			mustHave := []string{"fit-original", "tcx"}
			for _, format := range mustHave {
				if !containsFormat(links, format) {
					t.Errorf("Required export format %q missing from %s. Links: %v", format, testName, links)
				}
			}
		})
	}
}

// TestClientExportFormatsMatchActivityPage ties the format strings the
// runalyze client sends (GetFit, GetTcx) to the export links a real activity
// page advertises. If Runalyze renames a format (e.g. "fit" → "fit-original"),
// this fails loudly instead of silently 403-ing at download time.
func TestClientExportFormatsMatchActivityPage(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join("testdata", "fixtures", "activity-*.html"))
	if err != nil {
		t.Fatalf("failed to glob activity fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Skip("no activity fixture - run 'just update-fixtures'")
	}

	html, err := os.ReadFile(fixtures[0])
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	links := ExtractExportLinks(html)
	if len(links) == 0 {
		t.Fatalf("fixture %s contains no export links", fixtures[0])
	}

	cases := []struct {
		name   string
		format string
	}{
		{"FIT", runalyze.FitFormat},
		{"TCX", runalyze.TcxFormat},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantSuffix := "/export/file/" + c.format
			for _, link := range links {
				if strings.HasSuffix(link, wantSuffix) {
					return
				}
			}
			t.Errorf("client %s format %q not advertised by Runalyze activity page.\nExpected a link ending in %q.\nLinks in fixture: %v", c.name, c.format, wantSuffix, links)
		})
	}
}

func containsFormat(links []string, format string) bool {
	for _, link := range links {
		if strings.HasSuffix(link, "/export/file/"+format) {
			return true
		}
	}
	return false
}

// TestExtractExportLinks_Inline exercises the parser on a minimal HTML snippet
// (the download submenu shape) so the parser is covered even without fixtures.
func TestExtractExportLinks_Inline(t *testing.T) {
	html := []byte(`
<ul class="submenu">
  <li><a href="/activity/173659836/export/file/fit-original">as FIT</a></li>
  <li><a href="/activity/173659836/export/file/tcx">as TCX</a></li>
  <li><a href="/activity/173659836/export/file/gpx">as GPX</a></li>
  <li><a href="/activity/173659836/export/file/csv">as CSV</a></li>
  <li><a href="/activity/173659836/export/file/kml">as KML</a></li>
  <li><a href="/activity/173659836/export/file/fitlog">as FITLOG</a></li>
</ul>`)

	links := ExtractExportLinks(html)

	want := []string{
		"/activity/173659836/export/file/fit-original",
		"/activity/173659836/export/file/tcx",
		"/activity/173659836/export/file/gpx",
		"/activity/173659836/export/file/csv",
		"/activity/173659836/export/file/kml",
		"/activity/173659836/export/file/fitlog",
	}

	if len(links) != len(want) {
		t.Fatalf("got %d links, want %d: %v", len(links), len(want), links)
	}
	for i, got := range links {
		if got != want[i] {
			t.Errorf("link %d: got %q, want %q", i, got, want[i])
		}
	}
}
