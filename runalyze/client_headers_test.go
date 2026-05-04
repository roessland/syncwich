package runalyze

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGetDataBrowser_HeadersMimicBrowser asserts the XHR request shape matches
// a real Chrome XHR captured from runalyze.com/databrowser. Diverging from this
// shape triggers the upstream WAF and we get 502 Bad Gateway in production.
func TestGetDataBrowser_HeadersMimicBrowser(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		// Add User-Agent explicitly — net/http strips it from r.Header.
		got.Set("User-Agent", r.UserAgent())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html></html>"))
	}))
	defer srv.Close()

	prev := baseURL
	baseURL = srv.URL
	defer func() { baseURL = prev }()

	client := newTestClient(t)

	if _, err := client.GetDataBrowser(time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("GetDataBrowser failed: %v", err)
	}

	wantPresent := map[string]string{
		"User-Agent":         "Chrome/", // must contain a real-looking UA
		"Accept":             "text/html, */*; q=0.01",
		"Accept-Language":    "en-GB",
		"Sec-Ch-Ua":          "Chromium",
		"Sec-Ch-Ua-Mobile":   "?0",
		"Sec-Ch-Ua-Platform": `"macOS"`,
		"Sec-Fetch-Dest":     "empty",
		"Sec-Fetch-Mode":     "cors",
		"Sec-Fetch-Site":     "same-origin",
		"X-Requested-With":   "XMLHttpRequest",
		"Referer":            "/dashboard",
		"Dnt":                "1",
		"Priority":           "u=1",
	}
	for k, want := range wantPresent {
		v := got.Get(k)
		if v == "" {
			t.Errorf("missing header %q", k)
			continue
		}
		if !strings.Contains(v, want) {
			t.Errorf("header %q = %q, want substring %q", k, v, want)
		}
	}

	// These document-only headers must NOT be sent on an XHR.
	wantAbsent := []string{"Sec-Fetch-User", "Upgrade-Insecure-Requests"}
	for _, k := range wantAbsent {
		if v := got.Get(k); v != "" {
			t.Errorf("header %q must be absent on XHR, got %q", k, v)
		}
	}

	// User-Agent must not be Go's default.
	if ua := got.Get("User-Agent"); strings.HasPrefix(ua, "Go-http-client") {
		t.Errorf("User-Agent looks like Go default: %q", ua)
	}
}

func newTestClient(t *testing.T) *Client {
	t.Helper()
	cookiePath := filepath.Join(t.TempDir(), "cookies.json")
	c, err := New("u", "p", cookiePath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}
