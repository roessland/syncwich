package runalyze

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const (
	// FitFormat and TcxFormat are the export-URL format segments Runalyze
	// serves from /activity/{id}/export/file/{format}. Kept exported so
	// tests can cross-check them against a real activity page's link set.
	FitFormat = "fit-original"
	TcxFormat = "tcx"
)

// baseURL is a var (not const) so tests can point the client at httptest.
var baseURL = "https://runalyze.com"

var (
	// commonHeaders are sent on every request regardless of mode. Per-call
	// helpers layer on document- vs XHR-specific headers (sec-fetch-*,
	// accept, etc.) — keep those out of here.
	commonHeaders = map[string]string{
		"user-agent":         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36",
		"accept-language":    "en-GB,en;q=0.9,nb-NO;q=0.8,nb;q=0.7,sv-SE;q=0.6,sv;q=0.5,en-US;q=0.4",
		"sec-ch-ua":          `"Google Chrome";v="147", "Not.A/Brand";v="8", "Chromium";v="147"`,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": `"macOS"`,
		"dnt":                "1",
	}

	// Common errors
	ErrRedirectedToLogin = errors.New("redirected to login page")
)

// setDocumentHeaders mirrors what Chrome sends on a top-level navigation.
func setDocumentHeaders(req *http.Request) {
	for k, v := range commonHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-user", "?1")
	req.Header.Set("upgrade-insecure-requests", "1")
}

// setXHRHeaders mirrors what Chrome sends from a fetch()/XHR. The runalyze
// WAF returns 502 if document-only headers (sec-fetch-user,
// upgrade-insecure-requests) leak into an XHR, so this helper does NOT set
// them — callers must not add them either.
func setXHRHeaders(req *http.Request) {
	for k, v := range commonHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set("accept", "text/html, */*; q=0.01")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", baseURL+"/dashboard")
}

// Client represents a Runalyze API client
type Client struct {
	httpClient *http.Client
	username   string
	password   string
	cookiePath string
	logger     *log.Logger
	logLevel   string
}

// New creates a new Runalyze client
func New(username, password, cookiePath string) (*Client, error) {
	if cookiePath == "" {
		cookiePath = viper.GetString("cookie_path")
	}

	// Expand home directory if present
	expandedPath, err := homedir.Expand(cookiePath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand cookie path: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cookie directory: %w", err)
	}

	// Create logger first so we can pass it to the cookie jar
	logger := log.New(os.Stderr, "[runalyze] ", log.LstdFlags)
	logLevel := viper.GetString("log_level")

	// Create shouldLog function for the cookie jar
	shouldLogFn := func(level string) bool {
		levels := map[string]int{
			"trace": 0,
			"debug": 1,
			"info":  2,
			"warn":  3,
			"error": 4,
		}

		configuredLevel := logLevel
		if configuredLevel == "" {
			configuredLevel = "info" // Default to info if not set
		}

		return levels[level] >= levels[configuredLevel]
	}

	// Create persistent cookie jar with logger
	jar, err := newPersistentCookieJar(expandedPath, logger, shouldLogFn)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Clone the stdlib default transport so we inherit proxy-from-env,
	// connection pooling, and dial defaults — only override what we need.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	insecure := insecureTLSEnabled()
	transport.TLSClientConfig = buildTLSConfig(insecure)
	if insecure {
		logger.Printf("WARNING: SW_INSECURE_TLS is set — TLS certificate verification is DISABLED")
	}

	client := &http.Client{
		Transport: transport,
		Jar:       jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	return &Client{
		httpClient: client,
		username:   username,
		password:   password,
		cookiePath: expandedPath,
		logger:     logger,
		logLevel:   logLevel,
	}, nil
}

// shouldLog returns true if the given log level should be logged based on the configured log level
func (c *Client) shouldLog(level string) bool {
	levels := map[string]int{
		"trace": 0,
		"debug": 1,
		"info":  2,
		"warn":  3,
		"error": 4,
	}

	configuredLevel := c.logLevel
	if configuredLevel == "" {
		configuredLevel = "info" // Default to info if not set
	}

	return levels[level] >= levels[configuredLevel]
}

// logRequest logs the request details if log level is trace
func (c *Client) logRequest(req *http.Request, body []byte) {
	if !c.shouldLog("trace") {
		return
	}
	c.logger.Printf("Request Headers:")
	for k, v := range req.Header {
		c.logger.Printf("  %s: %s", k, strings.Join(v, ", "))
	}
	if len(body) > 0 {
		c.logger.Printf("Request Body: %s", string(body))
	}
}

// logResponse logs the response details if log level is trace
func (c *Client) logResponse(resp *http.Response, body []byte) {
	if !c.shouldLog("trace") {
		return
	}
	c.logger.Printf("Response Headers:")
	for k, v := range resp.Header {
		c.logger.Printf("  %s: %s", k, strings.Join(v, ", "))
	}
	if len(body) > 0 {
		preview := string(body)
		if len(preview) > 512 {
			preview = preview[:512]
		}
		c.logger.Printf("Response Body Preview: %s", preview)
	}
}

// doRequest performs an HTTP request with logging
func (c *Client) doRequest(req *http.Request) (*http.Response, []byte, error) {
	// Log request method and URL at debug level
	if c.shouldLog("debug") {
		c.logger.Printf("Request: %s %s", req.Method, req.URL)
	}

	// Log request
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	c.logRequest(req, bodyBytes)

	// Do request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Log response status at debug level
	if c.shouldLog("debug") {
		c.logger.Printf("Response: %s %s", resp.Status, req.URL)
	}

	// Read and log response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	c.logResponse(resp, respBody)

	return resp, respBody, nil
}

// Login performs the login process
func (c *Client) Login() error {
	csrfToken, err := c.doGetLogin()
	if err != nil {
		return fmt.Errorf("failed to get login page: %w", err)
	}

	err = c.doPostLogin(csrfToken)
	if err != nil {
		return fmt.Errorf("failed to post login: %w", err)
	}

	return nil
}

// doGetLogin retrieves the login page and extracts the CSRF token
func (c *Client) doGetLogin() (string, error) {
	req, err := http.NewRequest("GET", baseURL+"/login", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	setDocumentHeaders(req)

	resp, body, err := c.doRequest(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Extract CSRF token using regex
	re := regexp.MustCompile(`name="_csrf_token" value="([^"]+)"`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("csrf token not found in response")
	}

	return matches[1], nil
}

// doPostLogin performs the login POST request
func (c *Client) doPostLogin(csrfToken string) error {
	data := url.Values{}
	data.Set("_username", c.username)
	data.Set("_password", c.password)
	data.Set("_remember_me", "on")
	data.Set("submit", "Sign in")
	data.Set("_csrf_token", csrfToken)

	req, err := http.NewRequest("POST", baseURL+"/login", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	setDocumentHeaders(req)
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("cache-control", "max-age=0")

	resp, _, err := c.doRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetDataBrowser retrieves data for a specific week
func (c *Client) GetDataBrowser(startOfWeek time.Time) ([]byte, error) {
	// Calculate end of week (last second of Sunday)
	endOfWeek := startOfWeek.AddDate(0, 0, 7-int(startOfWeek.Weekday()))
	endOfWeek = time.Date(endOfWeek.Year(), endOfWeek.Month(), endOfWeek.Day(), 23, 59, 59, 0, time.UTC)

	startUnix := startOfWeek.UTC().Unix()
	endUnix := endOfWeek.UTC().Unix()

	url := fmt.Sprintf("%s/databrowser?start=%d&end=%d", baseURL, startUnix, endUnix)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	setXHRHeaders(req)

	resp, body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		if strings.HasSuffix(location, "/login") {
			return nil, ErrRedirectedToLogin
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return body, nil
}

// getActivityExport retrieves an export file for a specific activity ID and format
func (c *Client) getActivityExport(activityID, format string) ([]byte, string, error) {
	url := fmt.Sprintf("%s/activity/%s/export/file/%s", baseURL, activityID, format)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	setDocumentHeaders(req)
	req.Header.Set("referer", baseURL+"/dashboard")

	resp, body, err := c.doRequest(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Extract filename from content-disposition header
	contentDisposition := resp.Header.Get("content-disposition")
	if contentDisposition == "" {
		return nil, "", fmt.Errorf("content-disposition header not found")
	}

	// Parse filename from header
	re := regexp.MustCompile(`filename="([^"]+)"`)
	matches := re.FindStringSubmatch(contentDisposition)
	if len(matches) < 2 {
		return nil, "", fmt.Errorf("filename not found in content-disposition header")
	}

	return body, matches[1], nil
}

// GetFit retrieves a FIT file for a specific activity ID
func (c *Client) GetFit(activityID string) ([]byte, string, error) {
	return c.getActivityExport(activityID, FitFormat)
}

// GetTcx retrieves a TCX file for a specific activity ID
func (c *Client) GetTcx(activityID string) ([]byte, string, error) {
	return c.getActivityExport(activityID, TcxFormat)
}

// GetActivityPage retrieves the HTML of an activity's detail page.
// Used to scrape the export submenu so tests can verify the URL scheme
// hasn't drifted.
func (c *Client) GetActivityPage(activityID string) ([]byte, error) {
	url := fmt.Sprintf("%s/activity/%s", baseURL, activityID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	setDocumentHeaders(req)

	resp, body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound {
		if strings.HasSuffix(resp.Header.Get("Location"), "/login") {
			return nil, ErrRedirectedToLogin
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return body, nil
}

// PersistCookies explicitly saves the current cookies to disk
func (c *Client) PersistCookies() error {
	// Cast the jar to our persistent cookie jar to access the save method
	if pjar, ok := c.httpClient.Jar.(*persistentCookieJar); ok {
		pjar.mu.Lock()
		defer pjar.mu.Unlock()
		return pjar.save()
	}
	return fmt.Errorf("cookie jar is not a persistent cookie jar")
}
