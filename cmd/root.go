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
// - A duration string (30d, 2w, 1y, 6m)
func parseSinceDate(sinceStr string) (time.Time, error) {
	// First, try to parse as a duration
	if duration, err := parseDuration(sinceStr); err == nil {
		// Calculate the date by subtracting the duration from now
		return time.Now().Add(-duration), nil
	}

	// If not a duration, try to parse as a date using the existing logic
	return parseUntilDate(sinceStr)
}

// downloadActivityFile downloads an activity file (FIT or TCX) for the given activity ID
func downloadActivityFile(client *runalyze.Client, activityID string, saveDir string) error {
	fitPath := filepath.Join(saveDir, activityID+".fit")
	tcxPath := filepath.Join(saveDir, activityID+".tcx")

	// Check if either file already exists
	if _, err := os.Stat(fitPath); err == nil {
		fmt.Printf("FIT file already exists: %s\n", fitPath)
		return nil
	}
	if _, err := os.Stat(tcxPath); err == nil {
		fmt.Printf("TCX file already exists: %s\n", tcxPath)
		return nil
	}

	// Try to download FIT file first
	fitData, _, err := client.GetFit(activityID)
	if err != nil {
		// Check if it's a 404 error by looking at the error message or HTTP status
		if isNotFoundError(err) {
			fmt.Printf("FIT file not available for activity %s, trying TCX...\n", activityID)

			// Try to download TCX file
			tcxData, _, err := client.GetTcx(activityID)
			if err != nil {
				if isNotFoundError(err) {
					fmt.Printf("Neither FIT nor TCX file available for activity %s: https://runalyze.com/activity/%s\n", activityID, activityID)
					return nil // Continue to next activity
				}
				return fmt.Errorf("failed to download TCX file for activity %s: %w", activityID, err)
			}

			// Save TCX file
			if err := os.WriteFile(tcxPath, tcxData, 0644); err != nil {
				return fmt.Errorf("failed to save TCX file for activity %s: %w", activityID, err)
			}
			fmt.Printf("Saved TCX file: %s\n", tcxPath)
			time.Sleep(300 * time.Millisecond)
			return nil
		}
		return fmt.Errorf("failed to download FIT file for activity %s: %w", activityID, err)
	}

	// Save FIT file
	if err := os.WriteFile(fitPath, fitData, 0644); err != nil {
		return fmt.Errorf("failed to save FIT file for activity %s: %w", activityID, err)
	}
	fmt.Printf("Saved FIT file: %s\n", fitPath)
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

		// Set up cookie path with new default
		if cookiePath == "" {
			cookiePath = viper.GetString("cookie_path")
		}

		// Create client
		client, err := runalyze.New(username, password, cookiePath)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		// Try to get data to verify login
		_, err = client.GetDataBrowser(time.Now())
		if err != nil {
			// If we got redirected to login, try to login and retry
			if errors.Is(err, runalyze.ErrRedirectedToLogin) {
				if err := client.Login(); err != nil {
					return fmt.Errorf("failed to login: %w", err)
				}
				// Retry getting data after successful login
				_, err = client.GetDataBrowser(time.Now())
				if err != nil {
					return fmt.Errorf("failed to get data after login: %w", err)
				}
				// Persist cookies immediately after successful login verification
				if err := client.PersistCookies(); err != nil {
					return fmt.Errorf("failed to persist cookies after login: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get data: %w", err)
			}
		} else {
			// Persist cookies immediately after successful verification with existing cookies
			if err := client.PersistCookies(); err != nil {
				return fmt.Errorf("failed to persist cookies: %w", err)
			}
		}

		// Parse until date
		var until time.Time
		if untilStr != "" {
			until, err = parseUntilDate(untilStr)
			if err != nil {
				return fmt.Errorf("failed to parse until date: %w", err)
			}
		} else {
			// Default to current time, transformed to next Monday
			until = time.Now()
			daysUntilMonday := (8 - int(until.Weekday())) % 7
			until = until.AddDate(0, 0, daysUntilMonday)
			until = time.Date(until.Year(), until.Month(), until.Day(), 0, 0, 0, 0, until.Location())
		}

		// Parse since date
		var since time.Time
		if sinceStr != "" {
			since, err = parseSinceDate(sinceStr)
			if err != nil {
				return fmt.Errorf("failed to parse since date: %w", err)
			}
		} else {
			// Default to 4 weeks before the until date
			defaultDuration, _ := parseDuration("4w")
			since = until.Add(-defaultDuration)
		}

		// Validate that since is before until
		if since.After(until) || since.Equal(until) {
			return fmt.Errorf("--since date (%s) must be before --until date (%s)", since.Format("2006-01-02"), until.Format("2006-01-02"))
		}

		// Create an iterator starting from the specified Monday
		iter := rd.NewActivityIteratorWithSince(client, until, since)

		// Get save directory from config
		saveDir := viper.GetString("save_dir")

		// Expand save directory path
		expandedSaveDir, err := homedir.Expand(saveDir)
		if err != nil {
			return fmt.Errorf("failed to expand save directory path: %w", err)
		}

		// Create save directory if it doesn't exist
		if err := os.MkdirAll(expandedSaveDir, 0755); err != nil {
			return fmt.Errorf("failed to create save directory: %w", err)
		}

		fmt.Printf("Downloading activities from %s to %s\n", since.Format("2006-01-02"), until.Format("2006-01-02"))

		// Iterate through activities
		for activityID, ok := iter.Next(); ok; activityID, ok = iter.Next() {
			fmt.Printf("Found activity: %s\n", activityID)

			// Download activity file
			if err := downloadActivityFile(client, activityID, expandedSaveDir); err != nil {
				fmt.Printf("Error downloading activity %s: %v\n", activityID, err)
				continue
			}
		}

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

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
