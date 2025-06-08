package rd

import (
	"fmt"
	"path/filepath"
	"time"
)

// DownloadService handles the core download logic without presentation concerns
type DownloadService struct {
	client RunalyzeClient
	fs     FileSystem
	logger Logger
}

// NewDownloadService creates a new download service
func NewDownloadService(client RunalyzeClient, fs FileSystem, logger Logger) *DownloadService {
	return &DownloadService{
		client: client,
		fs:     fs,
		logger: logger,
	}
}

// DownloadActivity downloads a single activity and returns structured results
func (ds *DownloadService) DownloadActivity(activity ActivityInfo, saveDir string) DownloadResult {
	fitPath := filepath.Join(saveDir, activity.ID+".fit")
	tcxPath := filepath.Join(saveDir, activity.ID+".tcx")

	// Check if either file already exists
	if ds.fs.Exists(fitPath) {
		return DownloadResult{
			ActivityID: activity.ID,
			Success:    true,
			FileType:   "FIT",
			FilePath:   fitPath,
			Existed:    true,
		}
	}
	if ds.fs.Exists(tcxPath) {
		return DownloadResult{
			ActivityID: activity.ID,
			Success:    true,
			FileType:   "TCX",
			FilePath:   tcxPath,
			Existed:    true,
		}
	}

	// Try to download FIT file first
	fitData, _, err := ds.client.GetFit(activity.ID)
	if err != nil {
		// Check if it's a 404 error
		if isNotFoundError(err) {
			// FIT failed, try TCX
			tcxData, _, err := ds.client.GetTcx(activity.ID)
			if err != nil {
				if isNotFoundError(err) {
					// Neither available
					return DownloadResult{
						ActivityID: activity.ID,
						Success:    false,
						FileType:   "NONE",
						Error:      fmt.Errorf("neither FIT nor TCX available for activity %s", activity.ID),
					}
				}
				// Other TCX error
				return DownloadResult{
					ActivityID: activity.ID,
					Success:    false,
					FileType:   "TCX",
					Error:      fmt.Errorf("failed to download TCX file for activity %s: %w", activity.ID, err),
				}
			}

			// Save TCX file
			if err := ds.fs.WriteFile(tcxPath, tcxData, 0644); err != nil {
				return DownloadResult{
					ActivityID: activity.ID,
					Success:    false,
					FileType:   "TCX",
					Error:      fmt.Errorf("failed to save TCX file for activity %s: %w", activity.ID, err),
				}
			}

			return DownloadResult{
				ActivityID: activity.ID,
				Success:    true,
				FileType:   "TCX",
				FilePath:   tcxPath,
			}
		}

		// Other FIT error
		return DownloadResult{
			ActivityID: activity.ID,
			Success:    false,
			FileType:   "FIT",
			Error:      fmt.Errorf("failed to download FIT file for activity %s: %w", activity.ID, err),
		}
	}

	// Save FIT file
	if err := ds.fs.WriteFile(fitPath, fitData, 0644); err != nil {
		return DownloadResult{
			ActivityID: activity.ID,
			Success:    false,
			FileType:   "FIT",
			Error:      fmt.Errorf("failed to save FIT file for activity %s: %w", activity.ID, err),
		}
	}

	return DownloadResult{
		ActivityID: activity.ID,
		Success:    true,
		FileType:   "FIT",
		FilePath:   fitPath,
	}
}

// DownloadActivities downloads multiple activities and returns a summary
func (ds *DownloadService) DownloadActivities(iter *ActivityIterator, saveDir string) (*DownloadSummary, error) {
	// Ensure save directory exists
	if err := ds.fs.MkdirAll(saveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create save directory: %w", err)
	}

	var results []DownloadResult
	processedCount := 0
	errorCount := 0

	// Download all activities
	for activity, ok := iter.Next(); ok; activity, ok = iter.Next() {
		ds.logger.Debug("processing activity", "activity_id", activity.ID, "type", activity.Type)

		result := ds.DownloadActivity(activity, saveDir)
		results = append(results, result)

		processedCount++
		if !result.Success {
			errorCount++
		}

		// Small delay to be nice to the server
		if !result.Existed {
			time.Sleep(300 * time.Millisecond)
		}
	}

	return &DownloadSummary{
		Processed: processedCount,
		Errors:    errorCount,
		Results:   results,
	}, nil
}
