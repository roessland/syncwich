package runalyze

import (
	"bytes"
	"crypto/tls"
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
	baseURL = "https://runalyze.com"
)

var (
	commonHeaders = map[string]string{
		"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"accept-language":           "en-GB,en;q=0.9,nb-NO;q=0.8,nb;q=0.7,sv-SE;q=0.6,sv;q=0.5,en-US;q=0.4",
		"sec-ch-ua":                 "\"Google Chrome\";v=\"137\", \"Chromium\";v=\"137\", \"Not/A)Brand\";v=\"24\"",
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-platform":        "\"macOS\"",
		"sec-fetch-dest":            "document",
		"sec-fetch-mode":            "navigate",
		"sec-fetch-site":            "same-origin",
		"sec-fetch-user":            "?1",
		"upgrade-insecure-requests": "1",
	}

	// Common errors
	ErrRedirectedToLogin = errors.New("redirected to login page")
)

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
		home, err := homedir.Dir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cookiePath = filepath.Join(home, "proj", "runalyzedump", "cookie.json")
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

	// Create persistent cookie jar
	jar, err := newPersistentCookieJar(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	return &Client{
		httpClient: client,
		username:   username,
		password:   password,
		cookiePath: expandedPath,
		logger:     log.New(os.Stderr, "[runalyze] ", log.LstdFlags),
		logLevel:   viper.GetString("log_level"),
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

	for k, v := range commonHeaders {
		req.Header.Set(k, v)
	}

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

	for k, v := range commonHeaders {
		req.Header.Set(k, v)
	}
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

	for k, v := range commonHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("accept", "text/html, */*; q=0.01")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")

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

// GetFit retrieves a FIT file for a specific activity ID
func (c *Client) GetFit(activityID string) ([]byte, string, error) {
	url := fmt.Sprintf("%s/activity/%s/export/file/fit", baseURL, activityID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range commonHeaders {
		req.Header.Set(k, v)
	}
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
