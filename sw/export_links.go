package sw

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ExtractExportLinks returns every `/activity/{id}/export/file/{format}` href
// it finds in the given HTML, in document order. Used to verify that Runalyze
// still advertises the export formats syncwich depends on.
func ExtractExportLinks(htmlContent []byte) []string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
	if err != nil {
		return nil
	}

	var links []string
	doc.Find(`a[href*="/export/file/"]`).Each(func(_ int, a *goquery.Selection) {
		href, ok := a.Attr("href")
		if !ok {
			return
		}
		if !strings.Contains(href, "/activity/") {
			return
		}
		links = append(links, href)
	})
	return links
}
