package output

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/pterm/pterm"
)

// OutputMode defines how the application should output information
type OutputMode string

const (
	ModeInteractive OutputMode = "interactive" // Pretty output for humans
	ModeJSON        OutputMode = "json"        // Structured JSON output
	ModeDaemon      OutputMode = "daemon"      // Minimal output, logs to file
)

// UserOutput handles user-facing output (progress, results, status)
type UserOutput interface {
	// Progress shows ongoing operations
	Progress(format string, args ...any)
	// Status shows important state changes
	Status(format string, args ...any)
	// Result shows final results/summaries
	Result(format string, args ...any)
	// Error shows user-facing errors
	Error(format string, args ...any)
	// JSON outputs structured data (only in JSON mode)
	JSON(data any) error
}

// Logger wraps slog.Logger with context-aware methods
type Logger interface {
	// Component returns a logger for a specific component
	Component(name string) Logger
	// With returns a logger with additional attributes
	With(args ...any) Logger

	// Standard log levels
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// OutputLogger handles both user output and structured logging
type OutputLogger struct {
	Logger
	jsonMode bool
}

// Config holds output configuration
type Config struct {
	Mode           OutputMode
	LogLevel       string
	LogFile        string
	ShowTimestamps bool
	ShowComponent  bool
	JSONPretty     bool
}

// DownloadState represents the state of a file download
type DownloadState int

const (
	StateExists DownloadState = iota
	StateDownloading
	StateDownloaded
	StateError
	StateNotAvailable // New state for files that don't exist on server
)

// FileInfo represents information about a downloaded file
type FileInfo struct {
	Type     string // "FIT" or "TCX"
	State    DownloadState
	Progress int // 0-100
}

// MultiFileInfo represents information about multiple file attempts for one activity
type MultiFileInfo struct {
	Primary   FileInfo  // The main file being shown
	Secondary *FileInfo // Optional secondary file attempt (e.g., failed FIT when downloading TCX)
}

// New creates a new OutputLogger
// If jsonMode is true, only structured logs go to stdout
// If jsonMode is false, structured logs go to file and user messages use pterm
func New(jsonMode bool) (*OutputLogger, error) {
	var slogLogger *slog.Logger

	if jsonMode {
		// JSON mode: structured logs only to stdout
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: getLogLevel(),
		})
		slogLogger = slog.New(handler)
	} else {
		// Interactive mode: structured logs to file
		logFile, err := getLogFilePath()
		if err != nil {
			return nil, fmt.Errorf("failed to get log file path: %w", err)
		}

		// Create log directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		handler := slog.NewTextHandler(file, &slog.HandlerOptions{
			Level: getLogLevel(),
		})
		slogLogger = slog.New(handler)

		// pterm will automatically detect TTY and color support
	}

	logger := &loggerImpl{slog: slogLogger}

	return &OutputLogger{
		Logger:   logger,
		jsonMode: jsonMode,
	}, nil
}

// getLogLevel returns the log level from LOG_LEVEL env var, defaulting to debug
func getLogLevel() slog.Level {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "trace":
		return slog.LevelDebug - 4 // Trace is lower than debug
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug // Default to debug
	}
}

// getLogFilePath returns the path to the log file
func getLogFilePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".syncwich", "syncwich.log"), nil
}

// WeekHeader shows a week range header
func (ol *OutputLogger) WeekHeader(startDate, endDate time.Time) {
	if ol.jsonMode {
		ol.Logger.Info("week_start", "start_date", startDate.Format("2006-01-02"), "end_date", endDate.Format("2006-01-02"))
	} else {
		// Add a newline before the week header for proper spacing
		pterm.Println()
		headerText := fmt.Sprintf("📅 Week from %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		pterm.Info.Println(headerText)
	}
}

// ActivityLine shows a single activity line that can be updated
func (ol *OutputLogger) ActivityLine(emoji, activityID string, fileInfo FileInfo) *pterm.AreaPrinter {
	return ol.ActivityLineMulti(emoji, activityID, MultiFileInfo{Primary: fileInfo})
}

// ActivityLineMulti shows a single activity line with support for multiple file states
func (ol *OutputLogger) ActivityLineMulti(emoji, activityID string, multiFileInfo MultiFileInfo) *pterm.AreaPrinter {
	if ol.jsonMode {
		ol.Logger.Info("activity_status",
			"activity_id", activityID,
			"file_type", multiFileInfo.Primary.Type,
			"state", multiFileInfo.Primary.State,
			"progress", multiFileInfo.Primary.Progress)
		return nil
	}

	// Build the activity line with pterm styling
	line := ol.buildActivityLineMulti(emoji, activityID, multiFileInfo)

	// For existing files, just print and return nil
	if multiFileInfo.Primary.State == StateExists {
		pterm.Println(line)
		return nil
	}

	// Create an area printer for downloads that can be updated in place
	area, _ := pterm.DefaultArea.Start()
	area.Update(line)
	return area
}

// buildActivityLine creates a formatted activity line (legacy method)
func (ol *OutputLogger) buildActivityLine(emoji, activityID string, fileInfo FileInfo) string {
	return ol.buildActivityLineMulti(emoji, activityID, MultiFileInfo{Primary: fileInfo})
}

// buildActivityLineMulti creates a formatted activity line with support for multiple files
func (ol *OutputLogger) buildActivityLineMulti(emoji, activityID string, multiFileInfo MultiFileInfo) string {
	var parts []string

	// Add emoji and activity ID
	parts = append(parts, emoji, activityID)

	// Add secondary file if present (e.g., failed FIT attempt)
	if multiFileInfo.Secondary != nil {
		secondaryDisplay := ol.formatFileDisplay(*multiFileInfo.Secondary)
		parts = append(parts, secondaryDisplay)
	}

	// Add primary file
	primaryDisplay := ol.formatFileDisplay(multiFileInfo.Primary)
	parts = append(parts, primaryDisplay)

	// Add status
	statusDisplay := ol.formatStatusDisplay(multiFileInfo.Primary)
	parts = append(parts, statusDisplay)

	return strings.Join(parts, " ")
}

// formatFileDisplay formats a single file type with appropriate styling
func (ol *OutputLogger) formatFileDisplay(fileInfo FileInfo) string {
	switch fileInfo.State {
	case StateExists:
		return pterm.NewStyle(pterm.BgGray, pterm.FgBlack).Sprint(fileInfo.Type)
	case StateDownloading:
		return pterm.NewStyle(pterm.BgBlue, pterm.FgWhite).Sprint(fileInfo.Type)
	case StateDownloaded:
		return pterm.NewStyle(pterm.BgGreen, pterm.FgWhite).Sprint(fileInfo.Type)
	case StateError:
		return pterm.NewStyle(pterm.BgRed, pterm.FgWhite).Sprint(fileInfo.Type)
	case StateNotAvailable:
		return pterm.NewStyle(pterm.FgGray).Sprintf("%s (not available)", fileInfo.Type)
	default:
		return fileInfo.Type
	}
}

// formatStatusDisplay formats the status part of the line
func (ol *OutputLogger) formatStatusDisplay(fileInfo FileInfo) string {
	switch fileInfo.State {
	case StateExists:
		return pterm.NewStyle(pterm.FgGreen).Sprint("✅ Already downloaded")
	case StateDownloading:
		return fmt.Sprintf("Downloading... %d%%", fileInfo.Progress)
	case StateDownloaded:
		return pterm.NewStyle(pterm.FgGreen).Sprint("✅ Downloaded")
	case StateError:
		return pterm.NewStyle(pterm.FgRed).Sprint("❌ Error")
	case StateNotAvailable:
		return pterm.NewStyle(pterm.FgRed).Sprint("❌ Not available")
	default:
		return ""
	}
}

// UpdateActivityLine updates an existing activity line
func (ol *OutputLogger) UpdateActivityLine(area *pterm.AreaPrinter, emoji, activityID string, fileInfo FileInfo) {
	ol.UpdateActivityLineMulti(area, emoji, activityID, MultiFileInfo{Primary: fileInfo})
}

// UpdateActivityLineMulti updates an existing activity line with multi-file support
func (ol *OutputLogger) UpdateActivityLineMulti(area *pterm.AreaPrinter, emoji, activityID string, multiFileInfo MultiFileInfo) {
	if ol.jsonMode || area == nil {
		// In JSON mode, just log the update
		if ol.jsonMode {
			ol.Logger.Info("activity_update",
				"activity_id", activityID,
				"file_type", multiFileInfo.Primary.Type,
				"state", multiFileInfo.Primary.State,
				"progress", multiFileInfo.Primary.Progress)
		}
		return
	}

	// Build the updated line
	line := ol.buildActivityLineMulti(emoji, activityID, multiFileInfo)

	// Update the area with new content
	area.Update(line)

	// If download is complete or error, stop the area printer and add newline
	if multiFileInfo.Primary.State == StateDownloaded || multiFileInfo.Primary.State == StateError || multiFileInfo.Primary.State == StateNotAvailable {
		area.Stop()
		pterm.Println() // Add newline after stopping area printer
	}
}

// Progress shows ongoing operations (legacy method for backward compatibility)
func (ol *OutputLogger) Progress(format string, args ...any) {
	if ol.jsonMode {
		ol.Logger.Info("progress", "message", fmt.Sprintf(format, args...))
	} else {
		pterm.Info.Printf(format+"\n", args...)
	}
}

// Status shows important state changes
func (ol *OutputLogger) Status(format string, args ...any) {
	if ol.jsonMode {
		ol.Logger.Info("status", "message", fmt.Sprintf(format, args...))
	} else {
		pterm.Success.Printf(format+"\n", args...)
	}
}

// Result shows final results/summaries
func (ol *OutputLogger) Result(format string, args ...any) {
	if ol.jsonMode {
		ol.Logger.Info("result", "message", fmt.Sprintf(format, args...))
	} else {
		pterm.Success.Printf("🎯 "+format+"\n", args...)
	}
}

// Error shows user-facing errors
func (ol *OutputLogger) Error(format string, args ...any) {
	if ol.jsonMode {
		ol.Logger.Error("user_error", "message", fmt.Sprintf(format, args...))
	} else {
		pterm.Error.Printf(format+"\n", args...)
	}
}

// JSON outputs structured data (only in JSON mode)
func (ol *OutputLogger) JSON(data any) error {
	if !ol.jsonMode {
		return nil // Don't output JSON in interactive mode
	}

	// In JSON mode, output structured data directly to stdout
	encoder := json.NewEncoder(os.Stdout)
	return encoder.Encode(data)
}

// LogAndShowError logs an error with full context and shows a user-friendly message
func (ol *OutputLogger) LogAndShowError(err error, userMsg string, args ...any) {
	// Log the full error with context
	ol.Logger.Error("operation_failed", "error", err.Error(), "user_message", fmt.Sprintf(userMsg, args...))

	// Show user-friendly message
	ol.Error(userMsg, args...)
}

// loggerImpl implements Logger interface
type loggerImpl struct {
	slog *slog.Logger
}

func (l *loggerImpl) Component(name string) Logger {
	return &loggerImpl{slog: l.slog.With("component", name)}
}

func (l *loggerImpl) With(args ...any) Logger {
	return &loggerImpl{slog: l.slog.With(args...)}
}

func (l *loggerImpl) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

func (l *loggerImpl) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

func (l *loggerImpl) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

func (l *loggerImpl) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

// interactiveOutput provides pretty output for human users
type interactiveOutput struct {
	writer io.Writer
}

func (o *interactiveOutput) Progress(format string, args ...any) {
	fmt.Fprintf(o.writer, "⏳ "+format+"\n", args...)
}

func (o *interactiveOutput) Status(format string, args ...any) {
	fmt.Fprintf(o.writer, "ℹ️  "+format+"\n", args...)
}

func (o *interactiveOutput) Result(format string, args ...any) {
	fmt.Fprintf(o.writer, "✅ "+format+"\n", args...)
}

func (o *interactiveOutput) Error(format string, args ...any) {
	fmt.Fprintf(o.writer, "❌ "+format+"\n", args...)
}

func (o *interactiveOutput) JSON(data any) error {
	// In interactive mode, we don't output JSON
	return nil
}

// jsonOutput provides structured JSON output
type jsonOutput struct {
	writer io.Writer
	pretty bool
}

func (o *jsonOutput) Progress(format string, args ...any) {
	o.writeJSON(map[string]any{
		"type":      "progress",
		"message":   fmt.Sprintf(format, args...),
		"timestamp": time.Now().Unix(),
	})
}

func (o *jsonOutput) Status(format string, args ...any) {
	o.writeJSON(map[string]any{
		"type":      "status",
		"message":   fmt.Sprintf(format, args...),
		"timestamp": time.Now().Unix(),
	})
}

func (o *jsonOutput) Result(format string, args ...any) {
	o.writeJSON(map[string]any{
		"type":      "result",
		"message":   fmt.Sprintf(format, args...),
		"timestamp": time.Now().Unix(),
	})
}

func (o *jsonOutput) Error(format string, args ...any) {
	o.writeJSON(map[string]any{
		"type":      "error",
		"message":   fmt.Sprintf(format, args...),
		"timestamp": time.Now().Unix(),
	})
}

func (o *jsonOutput) JSON(data any) error {
	return o.writeJSON(data)
}

func (o *jsonOutput) writeJSON(data any) error {
	var encoder *json.Encoder
	encoder = json.NewEncoder(o.writer)
	if o.pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

// daemonOutput provides minimal output for daemon mode
type daemonOutput struct {
	writer io.Writer
}

func (o *daemonOutput) Progress(format string, args ...any) {
	// Minimal output in daemon mode
}

func (o *daemonOutput) Status(format string, args ...any) {
	fmt.Fprintf(o.writer, fmt.Sprintf(format, args...)+"\n")
}

func (o *daemonOutput) Result(format string, args ...any) {
	fmt.Fprintf(o.writer, fmt.Sprintf(format, args...)+"\n")
}

func (o *daemonOutput) Error(format string, args ...any) {
	fmt.Fprintf(o.writer, "ERROR: "+fmt.Sprintf(format, args...)+"\n")
}

func (o *daemonOutput) JSON(data any) error {
	return json.NewEncoder(o.writer).Encode(data)
}
