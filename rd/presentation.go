package rd

import (
	"time"

	"github.com/roessland/runalyzedump/pkg/output"
)

// PresentationService handles all presentation logic
type PresentationService struct {
	ol *output.OutputLogger
}

// NewPresentationService creates a new presentation service
func NewPresentationService(ol *output.OutputLogger) *PresentationService {
	return &PresentationService{ol: ol}
}

// ShowProgress displays a progress message
func (ps *PresentationService) ShowProgress(msg string) {
	ps.ol.Progress(msg)
}

// ShowStatus displays a status message
func (ps *PresentationService) ShowStatus(msg string, args ...any) {
	ps.ol.Status(msg, args...)
}

// ShowError logs and displays an error
func (ps *PresentationService) ShowError(err error, msg string, args ...any) {
	ps.ol.LogAndShowError(err, msg, args...)
}

// ShowWeekHeader displays a week header
func (ps *PresentationService) ShowWeekHeader(weekStart, weekEnd time.Time) {
	ps.ol.WeekHeader(weekStart, weekEnd)
}

// ShowActivityResult displays the result of downloading an activity
func (ps *PresentationService) ShowActivityResult(activity ActivityInfo, result DownloadResult) {
	if result.Existed {
		// File already existed
		ps.ol.ActivityLine(activity.TypeEmoji, activity.ID, output.FileInfo{
			Type:  result.FileType,
			State: output.StateExists,
		})
		return
	}

	if !result.Success {
		if result.FileType == "NONE" {
			// Neither format available
			ps.ol.ActivityLine(activity.TypeEmoji, activity.ID, output.FileInfo{
				Type:  "FIT/TCX",
				State: output.StateNotAvailable,
			})
		} else {
			// Download or save error
			ps.ol.ActivityLine(activity.TypeEmoji, activity.ID, output.FileInfo{
				Type:  result.FileType,
				State: output.StateError,
			})
		}
		return
	}

	// Successfully downloaded
	ps.ol.ActivityLine(activity.TypeEmoji, activity.ID, output.FileInfo{
		Type:  result.FileType,
		State: output.StateDownloaded,
	})
}

// ShowFinalResults displays the final download summary
func (ps *PresentationService) ShowFinalResults(summary *DownloadSummary) {
	ps.ol.Result("Download complete: %d processed, %d errors", summary.Processed, summary.Errors)
}

// ShowJSONResults outputs structured JSON results
func (ps *PresentationService) ShowJSONResults(summary *DownloadSummary, jsonMode bool) {
	if jsonMode {
		ps.ol.JSON(map[string]any{
			"summary": map[string]int{
				"processed": summary.Processed,
				"errors":    summary.Errors,
			},
			"date_range": map[string]string{
				"since": summary.Since.Format("2006-01-02"),
				"until": summary.Until.Format("2006-01-02"),
			},
		})
	}
}
