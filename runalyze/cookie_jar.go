package runalyze

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// persistentCookieJar implements a cookie jar that persists cookies to disk
type persistentCookieJar struct {
	*cookiejar.Jar
	path string
	mu   sync.Mutex
}

// cookieEntry represents a single cookie entry for serialization
type cookieEntry struct {
	Name       string    `json:"name"`
	Value      string    `json:"value"`
	Domain     string    `json:"domain"`
	Path       string    `json:"path"`
	Expires    time.Time `json:"expires"`
	RawExpires string    `json:"raw_expires,omitempty"`
	MaxAge     int       `json:"max_age"`
	Secure     bool      `json:"secure"`
	HttpOnly   bool      `json:"http_only"`
	SameSite   int       `json:"same_site"`
	Raw        string    `json:"raw,omitempty"`
	Unparsed   []string  `json:"unparsed,omitempty"`
}

// newPersistentCookieJar creates a new persistent cookie jar
func newPersistentCookieJar(path string) (*persistentCookieJar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	pjar := &persistentCookieJar{
		Jar:  jar,
		path: path,
	}

	// Try to load existing cookies
	if err := pjar.load(); err != nil {
		// It's okay if the file doesn't exist
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load cookies: %w", err)
		}
	}

	return pjar, nil
}

// SetCookies implements the http.CookieJar interface
func (j *persistentCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.Jar.SetCookies(u, cookies)

	// Save cookies after setting them
	if err := j.save(); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to save cookies: %v\n", err)
	}
}

// load reads cookies from the file
func (j *persistentCookieJar) load() error {
	data, err := os.ReadFile(j.path)
	if err != nil {
		return err
	}

	var entries []cookieEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal cookies: %w", err)
	}

	// Convert entries back to cookies
	cookies := make([]*http.Cookie, len(entries))
	for i, entry := range entries {
		cookies[i] = &http.Cookie{
			Name:       entry.Name,
			Value:      entry.Value,
			Path:       entry.Path,
			Domain:     entry.Domain,
			Expires:    entry.Expires,
			RawExpires: entry.RawExpires,
			MaxAge:     entry.MaxAge,
			Secure:     entry.Secure,
			HttpOnly:   entry.HttpOnly,
			SameSite:   http.SameSite(entry.SameSite),
			Raw:        entry.Raw,
			Unparsed:   entry.Unparsed,
		}
	}

	// Set cookies for all domains
	urls := make(map[string]*url.URL)
	for _, cookie := range cookies {
		if cookie.Domain == "" {
			continue
		}
		domain := cookie.Domain
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			domain = "https://" + domain
		}
		if _, ok := urls[domain]; !ok {
			u, err := url.Parse(domain)
			if err != nil {
				continue
			}
			urls[domain] = u
		}
	}

	for _, u := range urls {
		var domainCookies []*http.Cookie
		for _, cookie := range cookies {
			if cookie.Domain == u.Host {
				domainCookies = append(domainCookies, cookie)
			}
		}
		j.Jar.SetCookies(u, domainCookies)
	}

	return nil
}

// save writes cookies to the file
func (j *persistentCookieJar) save() error {
	// Get all cookies from the jar
	entries := make([]cookieEntry, 0)
	for _, u := range []string{"https://runalyze.com"} {
		parsedURL, err := url.Parse(u)
		if err != nil {
			continue
		}
		cookies := j.Jar.Cookies(parsedURL)
		for _, cookie := range cookies {
			entries = append(entries, cookieEntry{
				Name:       cookie.Name,
				Value:      cookie.Value,
				Path:       cookie.Path,
				Domain:     cookie.Domain,
				Expires:    cookie.Expires,
				RawExpires: cookie.RawExpires,
				MaxAge:     cookie.MaxAge,
				Secure:     cookie.Secure,
				HttpOnly:   cookie.HttpOnly,
				SameSite:   int(cookie.SameSite),
				Raw:        cookie.Raw,
				Unparsed:   cookie.Unparsed,
			})
		}
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	if err := os.WriteFile(j.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write cookies: %w", err)
	}

	return nil
}
