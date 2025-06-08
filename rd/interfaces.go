package rd

import (
	"time"
)

// RunalyzeClient interface abstracts the Runalyze client for testing
type RunalyzeClient interface {
	GetFit(id string) ([]byte, string, error)
	GetTcx(id string) ([]byte, string, error)
	GetDataBrowser(date time.Time) ([]byte, error)
	Login() error
	PersistCookies() error
}

// FileSystem interface abstracts file operations for testing
type FileSystem interface {
	WriteFile(path string, data []byte, perm int) error
	Exists(path string) bool
	MkdirAll(path string, perm int) error
}

// Logger interface abstracts logging for testing
type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}

// DownloadResult represents the result of downloading a single activity
type DownloadResult struct {
	ActivityID string
	Success    bool
	FileType   string // "FIT", "TCX", or "NONE"
	FilePath   string
	Error      error
	Existed    bool // true if file already existed
}

// DownloadSummary represents the overall download results
type DownloadSummary struct {
	Processed int
	Errors    int
	Since     time.Time
	Until     time.Time
	Results   []DownloadResult
}
