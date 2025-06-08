package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/roessland/runalyzedump/pkg/output"
	"github.com/roessland/runalyzedump/rd"
	"github.com/roessland/runalyzedump/runalyze"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	username   string
	password   string
	cookiePath string
	cfgFile    string
	untilStr   string
	sinceStr   string
	jsonMode   bool
)

// parseUntilDate parses a date string in YYYY-MM-DD, YYYY-MM, or YYYY format and returns the next Monday
func parseUntilDate(dateStr string) (time.Time, error) {
	var t time.Time
	var err error

	// Try parsing as YYYY-MM-DD
	t, err = time.Parse("2006-01-02", dateStr)
	if err != nil {
		// Try parsing as YYYY-MM
		t, err = time.Parse("2006-01", dateStr)
		if err != nil {
			// Try parsing as YYYY
			t, err = time.Parse("2006", dateStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid date format. Use YYYY-MM-DD, YYYY-MM, or YYYY")
			}
			// If it's just a year, use the last day of the year (December 31st)
			t = time.Date(t.Year(), 12, 31, 0, 0, 0, 0, t.Location())
		} else {
			// If it's just a month, use the last day of the month
			t = time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location())
		}
	}

	// If it's already a Monday, return it
	if t.Weekday() == time.Monday {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
	}

	// Find the next Monday
	daysUntilMonday := (8 - int(t.Weekday())) % 7
	nextMonday := t.AddDate(0, 0, daysUntilMonday)
	return time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 0, 0, 0, 0, nextMonday.Location()), nil
}

// parseDuration parses a simplified prometheus-style duration string
// Supports: y (years), w (weeks), d (days), m (months)
// Examples: "30d", "2w", "1y", "6m"
// No combinations allowed (e.g., "1y2w" is invalid)
func parseDuration(durationStr string) (time.Duration, error) {
	// Regex to match the simplified duration format
	re := regexp.MustCompile(`^([0-9]+)([ywdm])$`)
	matches := re.FindStringSubmatch(durationStr)

	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid duration format. Use format like '30d', '2w', '1y', or '6m' (no combinations allowed)")
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", matches[1])
	}

	unit := matches[2]

	switch unit {
	case "y":
		// Approximate: 365 days per year
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "m":
		// Approximate: 30 days per month
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid duration unit: %s (use y, w, d, or m)", unit)
	}
}

// parseSinceDate parses a --since parameter which can be either:
// - A date string (YYYY-MM-DD, YYYY-MM, or YYYY format)
// - A duration string (30d, 2w, 1y, 6m) - relative to the until date
func parseSinceDate(sinceStr string, untilDate time.Time) (time.Time, error) {
	// First, try to parse as a duration
	if duration, err := parseDuration(sinceStr); err == nil {
		// Calculate the date by subtracting the duration from the until date
		return untilDate.Add(-duration), nil
	}

	// If not a duration, try to parse as a date using the existing logic
	return parseUntilDate(sinceStr)
}

// validateAndParseDates validates and parses the until and since date parameters early
func validateAndParseDates(untilStr, sinceStr string) (since, until time.Time, err error) {
	// Parse until date
	if untilStr != "" {
		until, err = parseUntilDate(untilStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse until date: %w", err)
		}
	} else {
		// Default to current time, transformed to next Monday
		until = time.Now()
		daysUntilMonday := (8 - int(until.Weekday())) % 7
		until = until.AddDate(0, 0, daysUntilMonday)
		until = time.Date(until.Year(), until.Month(), until.Day(), 0, 0, 0, 0, until.Location())
	}

	// Parse since date (relative to until date)
	if sinceStr != "" {
		since, err = parseSinceDate(sinceStr, until)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("failed to parse since date: %w", err)
		}
	} else {
		// Default to 4 weeks before the until date
		defaultDuration, _ := parseDuration("4w")
		since = until.Add(-defaultDuration)
	}

	// Validate that since is before until
	if since.After(until) || since.Equal(until) {
		return time.Time{}, time.Time{}, fmt.Errorf("--since date (%s) must be before --until date (%s)", since.Format("2006-01-02"), until.Format("2006-01-02"))
	}

	return since, until, nil
}

// downloadActivityFile downloads an activity file (FIT or TCX) for the given activity info
func downloadActivityFile(client *runalyze.Client, activity rd.ActivityInfo, saveDir string, ol *output.OutputLogger) error {
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

// isNotFoundError checks if the error indicates a 404 Not Found response
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check if the error message contains status code 404
	return fmt.Sprintf("%v", err) == "unexpected status code: 404"
}

var rootCmd = &cobra.Command{
	Use:   "runalyzedump",
	Short: "A tool to dump Runalyze data",
	Long:  `A tool to dump Runalyze data for analysis and backup purposes.`,
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download activities from Runalyze",
	Long:  `Download activities from Runalyze and save them as FIT or TCX files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate dates first, before any other operations
		since, until, err := validateAndParseDates(untilStr, sinceStr)
		if err != nil {
			return err // Return error directly for early validation
		}

		// Create output/logger system
		ol, err := output.New(jsonMode)
		if err != nil {
			return fmt.Errorf("failed to create output system: %w", err)
		}

		// Create component-specific logger
		logger := ol.Component("download")

		// Check for credentials
		if username == "" {
			username = viper.GetString("username")
		}
		if password == "" {
			password = viper.GetString("password")
		}

		if username == "" || password == "" {
			return fmt.Errorf("username and password must be provided via config file, environment variables, or command line flags")
		}

		logger.Info("starting download process", "username", username)

		// Set up cookie path with new default
		if cookiePath == "" {
			cookiePath = viper.GetString("cookie_path")
		}

		// Create client
		client, err := runalyze.New(username, password, cookiePath)
		if err != nil {
			ol.LogAndShowError(err, "Failed to create Runalyze client")
			return err
		}

		// Try to get data to verify login
		ol.Progress("Verifying login credentials...")
		logger.Debug("attempting to verify login")

		_, err = client.GetDataBrowser(time.Now())
		if err != nil {
			// If we got redirected to login, try to login and retry
			if errors.Is(err, runalyze.ErrRedirectedToLogin) {
				ol.Progress("Logging in to Runalyze...")
				logger.Info("attempting login")

				if err := client.Login(); err != nil {
					ol.LogAndShowError(err, "Failed to login to Runalyze")
					return err
				}
				// Retry getting data after successful login
				_, err = client.GetDataBrowser(time.Now())
				if err != nil {
					ol.LogAndShowError(err, "Failed to verify login after authentication")
					return err
				}
				// Persist cookies immediately after successful login verification
				if err := client.PersistCookies(); err != nil {
					logger.Warn("failed to persist cookies", "error", err)
				}
				ol.Status("Successfully logged in to Runalyze")
			} else {
				ol.LogAndShowError(err, "Failed to verify Runalyze connection")
				return err
			}
		} else {
			// Persist cookies immediately after successful verification with existing cookies
			ol.Status("Using existing Runalyze session")
			if err := client.PersistCookies(); err != nil {
				logger.Warn("failed to persist cookies", "error", err)
			}
		}

		logger.Info("download configuration",
			"since", since.Format("2006-01-02"),
			"until", until.Format("2006-01-02"))

		// Create an iterator starting from the specified Monday
		iter := rd.NewActivityIteratorWithSince(client, until, since)
		iter.SetLogger(logger)

		// Get save directory from config
		saveDir := viper.GetString("save_dir")

		// Expand save directory path
		expandedSaveDir, err := homedir.Expand(saveDir)
		if err != nil {
			ol.LogAndShowError(err, "Failed to expand save directory path")
			return err
		}

		// Create save directory if it doesn't exist
		if err := os.MkdirAll(expandedSaveDir, 0755); err != nil {
			ol.LogAndShowError(err, "Failed to create save directory: %s", expandedSaveDir)
			return err
		}

		ol.Status("Downloading activities from %s to %s", since.Format("2006-01-02"), until.Format("2006-01-02"))

		// Track counts for final summary
		var downloadedCount, errorCount int
		var currentWeekStart time.Time

		// Iterate through activities
		for activity, ok := iter.Next(); ok; activity, ok = iter.Next() {
			// Show week header when we encounter a new week
			if activity.WeekStart != currentWeekStart {
				currentWeekStart = activity.WeekStart
				ol.WeekHeader(activity.WeekStart, activity.WeekEnd)
			}

			logger.Debug("processing activity", "activity_id", activity.ID, "type", activity.Type)

			// Download activity file
			if err := downloadActivityFile(client, activity, expandedSaveDir, ol); err != nil {
				ol.LogAndShowError(err, "Error downloading activity %s", activity.ID)
				errorCount++
				continue
			}
			downloadedCount++
		}

		// Show final results
		ol.Result("Download complete: %d processed, %d errors", downloadedCount, errorCount)

		// Output structured results for JSON mode
		if jsonMode {
			ol.JSON(map[string]any{
				"summary": map[string]int{
					"processed": downloadedCount,
					"errors":    errorCount,
				},
				"date_range": map[string]string{
					"since": since.Format("2006-01-02"),
					"until": until.Format("2006-01-02"),
				},
			})
		}

		logger.Info("download completed",
			"processed", downloadedCount,
			"errors", errorCount)

		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Set default values using viper
	viper.SetDefault("save_dir", "~/.runalyzedump/activities")
	viper.SetDefault("cookie_path", "~/.runalyzedump/runalyze-cookie.json")

	// Add persistent flags to root command
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.runalyzedump/runalyzedump.yaml)")
	rootCmd.PersistentFlags().String("save_dir", "", "Directory to save downloaded files (default: ~/.runalyzedump/activities)")

	// Add flags specific to download command
	downloadCmd.Flags().StringVar(&username, "username", "", "Runalyze username")
	downloadCmd.Flags().StringVar(&password, "password", "", "Runalyze password")
	downloadCmd.Flags().StringVar(&cookiePath, "cookie-path", "", "Path to cookie file (default: ~/.runalyzedump/runalyze-cookie.json)")
	downloadCmd.Flags().StringVar(&untilStr, "until", "", "Date to start from (YYYY-MM-DD, YYYY-MM, or YYYY format).")
	downloadCmd.Flags().StringVar(&sinceStr, "since", "", "Date to stop at (YYYY-MM-DD, YYYY-MM, YYYY format) or duration ago (e.g., 30d, 2w, 1y, 6m). Default: 4w")
	downloadCmd.Flags().BoolVar(&jsonMode, "json", false, "Output structured JSON logs to stdout (for cron/systemd)")

	// Bind flags to environment variables
	viper.BindEnv("username", "RUNALYZE_USERNAME")
	viper.BindEnv("password", "RUNALYZE_PASSWORD")
	viper.BindEnv("cookie_path", "RUNALYZE_COOKIE_PATH")
	viper.BindEnv("save_dir", "RUNALYZE_SAVE_DIR")
	viper.BindPFlag("save_dir", rootCmd.PersistentFlags().Lookup("save_dir"))

	// Add download command to root
	rootCmd.AddCommand(downloadCmd)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in ~/.runalyzedump/ directory with name "runalyzedump" (without extension).
		viper.AddConfigPath(filepath.Join(home, ".runalyzedump"))
		viper.SetConfigName("runalyzedump")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in silently (logging is via LOG_LEVEL env var)
	viper.ReadInConfig()
}
