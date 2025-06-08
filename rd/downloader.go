package rd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/roessland/runalyzedump/pkg/output"
	"github.com/roessland/runalyzedump/runalyze"
)

// DownloadConfig holds all configuration needed for downloading activities
type DownloadConfig struct {
	Username   string
	Password   string
	CookiePath string
	UntilStr   string
	SinceStr   string
	SaveDir    string
	JSONMode   bool
}

// isNotFoundError checks if the error indicates a 404 Not Found response
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check if the error message contains status code 404
	return fmt.Sprintf("%v", err) == "unexpected status code: 404"
}

// downloadActivityFile downloads an activity file (FIT or TCX) for the given activity info
func downloadActivityFile(client *runalyze.Client, activity ActivityInfo, saveDir string, ol *output.OutputLogger) error {
	fitPath := filepath.Join(saveDir, activity.ID+".fit")
	tcxPath := filepath.Join(saveDir, activity.ID+".tcx")

	// Check if either file already exists
	if _, err := os.Stat(fitPath); err == nil {
		// File exists - show completed line
		ol.ActivityLine(activity.TypeEmoji, activity.ID, output.FileInfo{
			Type:  "FIT",
			State: output.StateExists,
		})
		return nil
	}
	if _, err := os.Stat(tcxPath); err == nil {
		// File exists - show completed line
		ol.ActivityLine(activity.TypeEmoji, activity.ID, output.FileInfo{
			Type:  "TCX",
			State: output.StateExists,
		})
		return nil
	}

	// Start with FIT download - create one area printer for this activity
	area := ol.ActivityLine(activity.TypeEmoji, activity.ID, output.FileInfo{
		Type:     "FIT",
		State:    output.StateDownloading,
		Progress: 0,
	})

	// Update progress to 50% (simulating headers received)
	if area != nil {
		ol.UpdateActivityLine(area, activity.TypeEmoji, activity.ID, output.FileInfo{
			Type:     "FIT",
			State:    output.StateDownloading,
			Progress: 50,
		})
	}

	// Try to download FIT file first
	fitData, _, err := client.GetFit(activity.ID)
	if err != nil {
		// Check if it's a 404 error by looking at the error message or HTTP status
		if isNotFoundError(err) {
			// FIT failed, try TCX - update to show TCX attempt while remembering FIT failure
			if area != nil {
				ol.UpdateActivityLine(area, activity.TypeEmoji, activity.ID, output.FileInfo{
					Type:     "TCX",
					State:    output.StateDownloading,
					Progress: 0,
				})
			}

			// Update progress to 50% for TCX
			if area != nil {
				ol.UpdateActivityLine(area, activity.TypeEmoji, activity.ID, output.FileInfo{
					Type:     "TCX",
					State:    output.StateDownloading,
					Progress: 50,
				})
			}

			// Try to download TCX file
			tcxData, _, err := client.GetTcx(activity.ID)
			if err != nil {
				if isNotFoundError(err) {
					// Neither available - show both failed attempts
					if area != nil {
						ol.UpdateActivityLineMulti(area, activity.TypeEmoji, activity.ID, output.MultiFileInfo{
							Primary:   output.FileInfo{Type: "TCX", State: output.StateNotAvailable},
							Secondary: &output.FileInfo{Type: "FIT", State: output.StateNotAvailable},
						})
					}
					return nil // Continue to next activity
				}
				// Other error - show FIT not available, TCX error
				if area != nil {
					ol.UpdateActivityLineMulti(area, activity.TypeEmoji, activity.ID, output.MultiFileInfo{
						Primary:   output.FileInfo{Type: "TCX", State: output.StateError},
						Secondary: &output.FileInfo{Type: "FIT", State: output.StateNotAvailable},
					})
				}
				return fmt.Errorf("failed to download TCX file for activity %s: %w", activity.ID, err)
			}

			// Update progress to 100% for TCX
			if area != nil {
				ol.UpdateActivityLine(area, activity.TypeEmoji, activity.ID, output.FileInfo{
					Type:     "TCX",
					State:    output.StateDownloading,
					Progress: 100,
				})
			}

			// Save TCX file
			if err := os.WriteFile(tcxPath, tcxData, 0644); err != nil {
				if area != nil {
					ol.UpdateActivityLineMulti(area, activity.TypeEmoji, activity.ID, output.MultiFileInfo{
						Primary:   output.FileInfo{Type: "TCX", State: output.StateError},
						Secondary: &output.FileInfo{Type: "FIT", State: output.StateNotAvailable},
					})
				}
				return fmt.Errorf("failed to save TCX file for activity %s: %w", activity.ID, err)
			}

			// Mark TCX as downloaded - show both FIT (not available) and TCX (downloaded)
			if area != nil {
				ol.UpdateActivityLineMulti(area, activity.TypeEmoji, activity.ID, output.MultiFileInfo{
					Primary:   output.FileInfo{Type: "TCX", State: output.StateDownloaded},
					Secondary: &output.FileInfo{Type: "FIT", State: output.StateNotAvailable},
				})
			}

			time.Sleep(300 * time.Millisecond)
			return nil
		}

		// Other FIT error - update same line to show error
		if area != nil {
			ol.UpdateActivityLine(area, activity.TypeEmoji, activity.ID, output.FileInfo{
				Type:  "FIT",
				State: output.StateError,
			})
		}
		return fmt.Errorf("failed to download FIT file for activity %s: %w", activity.ID, err)
	}

	// FIT download successful - update to 100%
	if area != nil {
		ol.UpdateActivityLine(area, activity.TypeEmoji, activity.ID, output.FileInfo{
			Type:     "FIT",
			State:    output.StateDownloading,
			Progress: 100,
		})
	}

	// Save FIT file
	if err := os.WriteFile(fitPath, fitData, 0644); err != nil {
		if area != nil {
			ol.UpdateActivityLine(area, activity.TypeEmoji, activity.ID, output.FileInfo{
				Type:  "FIT",
				State: output.StateError,
			})
		}
		return fmt.Errorf("failed to save FIT file for activity %s: %w", activity.ID, err)
	}

	// Mark FIT as downloaded - update same line
	if area != nil {
		ol.UpdateActivityLine(area, activity.TypeEmoji, activity.ID, output.FileInfo{
			Type:  "FIT",
			State: output.StateDownloaded,
		})
	}

	time.Sleep(300 * time.Millisecond)
	return nil
}

// Download performs the main download orchestration using the new service-based architecture
func Download(config DownloadConfig) error {
	// 1. Validate and parse dates
	since, until, err := ValidateAndParseDates(config.UntilStr, config.SinceStr)
	if err != nil {
		return err
	}

	// 2. Setup dependencies
	_, logger, presentation, err := setupDependencies(config)
	if err != nil {
		return err
	}

	// 3. Validate credentials
	if err := validateCredentials(config); err != nil {
		return err
	}

	logger.Info("starting download process", "username", config.Username)

	// 4. Create and authenticate client
	client, err := createAndAuthenticateClient(config, logger, presentation)
	if err != nil {
		return err
	}

	// 5. Setup services
	fs := NewOSFileSystem()
	downloadService := NewDownloadService(client, fs, logger)

	// 6. Prepare download directory
	expandedSaveDir, err := prepareDownloadDirectory(config.SaveDir, fs, presentation)
	if err != nil {
		return err
	}

	// 7. Download activities
	summary, err := downloadActivities(client, downloadService, presentation, since, until, expandedSaveDir, logger)
	if err != nil {
		return err
	}

	// 8. Show final results
	summary.Since = since
	summary.Until = until
	presentation.ShowFinalResults(summary)
	presentation.ShowJSONResults(summary, config.JSONMode)

	logger.Info("download completed",
		"processed", summary.Processed,
		"errors", summary.Errors)

	return nil
}

// setupDependencies creates the output logger and presentation service
func setupDependencies(config DownloadConfig) (*output.OutputLogger, Logger, *PresentationService, error) {
	ol, err := output.New(config.JSONMode)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create output system: %w", err)
	}

	logger := ol.Component("download")
	presentation := NewPresentationService(ol)

	return ol, logger, presentation, nil
}

// validateCredentials checks that username and password are provided
func validateCredentials(config DownloadConfig) error {
	if config.Username == "" || config.Password == "" {
		return fmt.Errorf("username and password must be provided via config file, environment variables, or command line flags")
	}
	return nil
}

// createAndAuthenticateClient creates a Runalyze client and ensures it's authenticated
func createAndAuthenticateClient(config DownloadConfig, logger Logger, presentation *PresentationService) (*runalyze.Client, error) {
	// Create client
	client, err := runalyze.New(config.Username, config.Password, config.CookiePath)
	if err != nil {
		presentation.ShowError(err, "Failed to create Runalyze client")
		return nil, err
	}

	// Authenticate
	presentation.ShowProgress("Verifying login credentials...")
	authService := NewAuthService(client, logger)

	if err := authService.EnsureAuthenticated(); err != nil {
		presentation.ShowError(err, "Failed to authenticate with Runalyze")
		return nil, err
	}

	presentation.ShowStatus("Successfully authenticated with Runalyze")
	return client, nil
}

// prepareDownloadDirectory expands and creates the download directory
func prepareDownloadDirectory(saveDir string, fs FileSystem, presentation *PresentationService) (string, error) {
	expandedSaveDir, err := homedir.Expand(saveDir)
	if err != nil {
		presentation.ShowError(err, "Failed to expand save directory path")
		return "", err
	}

	if err := fs.MkdirAll(expandedSaveDir, 0755); err != nil {
		presentation.ShowError(err, "Failed to create save directory: %s", expandedSaveDir)
		return "", err
	}

	return expandedSaveDir, nil
}

// downloadActivities orchestrates the download of all activities in the date range
func downloadActivities(client *runalyze.Client, downloadService *DownloadService, presentation *PresentationService, since, until time.Time, saveDir string, logger Logger) (*DownloadSummary, error) {
	logger.Info("download configuration",
		"since", since.Format("2006-01-02"),
		"until", until.Format("2006-01-02"))

	// Create an iterator starting from the specified Monday
	iter := NewActivityIteratorWithSince(client, until, since)
	iter.SetLogger(logger)

	presentation.ShowStatus("Downloading activities from %s to %s", since.Format("2006-01-02"), until.Format("2006-01-02"))

	// Track for week headers
	var currentWeekStart time.Time
	var results []DownloadResult
	processedCount := 0
	errorCount := 0

	// Download all activities with presentation
	for activity, ok := iter.Next(); ok; activity, ok = iter.Next() {
		// Show week header when we encounter a new week
		if activity.WeekStart != currentWeekStart {
			currentWeekStart = activity.WeekStart
			presentation.ShowWeekHeader(activity.WeekStart, activity.WeekEnd)
		}

		logger.Debug("processing activity", "activity_id", activity.ID, "type", activity.Type)

		// Download the activity
		result := downloadService.DownloadActivity(activity, saveDir)
		results = append(results, result)

		// Show the result
		presentation.ShowActivityResult(activity, result)

		processedCount++
		if !result.Success {
			errorCount++
		}

		// Small delay to be nice to the server (only if we actually downloaded)
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
