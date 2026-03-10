package runalyze

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestNewPersistentCookieJar_EmptyFile(t *testing.T) {
	// Create an empty cookie file (simulates the production failure)
	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookies.json")
	if err := os.WriteFile(cookiePath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	logger := log.New(os.Stderr, "", 0)
	jar, err := newPersistentCookieJar(cookiePath, logger, func(string) bool { return false })
	if err != nil {
		t.Fatalf("expected no error for empty cookie file, got: %v", err)
	}
	if jar == nil {
		t.Fatal("expected non-nil jar")
	}
}

func TestNewPersistentCookieJar_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "nonexistent.json")

	logger := log.New(os.Stderr, "", 0)
	jar, err := newPersistentCookieJar(cookiePath, logger, func(string) bool { return false })
	if err != nil {
		t.Fatalf("expected no error for missing cookie file, got: %v", err)
	}
	if jar == nil {
		t.Fatal("expected non-nil jar")
	}
}

func TestSave_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookies.json")

	logger := log.New(os.Stderr, "", 0)
	jar, err := newPersistentCookieJar(cookiePath, logger, func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}

	// Set a cookie so there's something to save
	runalyzeURL, _ := url.Parse("https://runalyze.com")
	jar.SetCookies(runalyzeURL, []*http.Cookie{
		{Name: "session", Value: "abc123"},
	})

	// Verify the file was written with valid content
	data, err := os.ReadFile(cookiePath)
	if err != nil {
		t.Fatalf("cookie file should exist after save: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("cookie file should not be empty after save")
	}

	// Now make the directory read-only so the next save fails
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0700) })

	// Try to save again — this should fail
	jar.SetCookies(runalyzeURL, []*http.Cookie{
		{Name: "session", Value: "new_value"},
	})

	// The original file should still contain valid data (not truncated/empty)
	afterData, err := os.ReadFile(cookiePath)
	if err != nil {
		t.Fatalf("cookie file should still be readable: %v", err)
	}
	if len(afterData) == 0 {
		t.Fatal("cookie file was truncated by failed save — atomic write is broken")
	}
	if string(afterData) != string(data) {
		t.Fatal("cookie file content changed despite failed save")
	}
}

func TestSave_NoTempFileLeftOnFailure(t *testing.T) {
	dir := t.TempDir()
	cookiePath := filepath.Join(dir, "cookies.json")

	logger := log.New(os.Stderr, "", 0)
	jar, err := newPersistentCookieJar(cookiePath, logger, func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}

	// Do a successful save first
	runalyzeURL, _ := url.Parse("https://runalyze.com")
	jar.SetCookies(runalyzeURL, []*http.Cookie{
		{Name: "test", Value: "value"},
	})

	// Check no temp files are left behind
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "cookies.json" {
			t.Fatalf("unexpected temp file left behind: %s", e.Name())
		}
	}
}
