package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
)

var rootCmd = &cobra.Command{
	Use:   "runalyzedump",
	Short: "A tool to dump Runalyze data",
	Long:  `A tool to dump Runalyze data for analysis and backup purposes.`,
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

		// Set up cookie path
		if cookiePath == "" {
			cookiePath = viper.GetString("cookie_path")
		}
		if cookiePath == "" {
			home, err := homedir.Dir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			cookiePath = filepath.Join(home, "proj", "runalyzedump", "cookie.json")
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
			} else {
				return fmt.Errorf("failed to get data: %w", err)
			}
		}

		// Create an iterator starting from the current week's Monday
		now := time.Now()
		monday := now.AddDate(0, 0, -int(now.Weekday())+1)
		monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
		iter := rd.NewActivityIterator(client, monday)

		// Iterate through activities
		for activityID, ok := iter.Next(); ok; activityID, ok = iter.Next() {
			fmt.Printf("Found activity: %s\n", activityID)

			// Get save directory from config
			saveDir := viper.GetString("save_dir")
			if saveDir == "" {
				saveDir = "~/proj/runalyzedump/output"
			}

			// Expand save directory path
			expandedSaveDir, err := homedir.Expand(saveDir)
			if err != nil {
				return fmt.Errorf("failed to expand save directory path: %w", err)
			}

			// Create save directory if it doesn't exist
			if err := os.MkdirAll(expandedSaveDir, 0755); err != nil {
				return fmt.Errorf("failed to create save directory: %w", err)
			}

			// Check if file already exists
			fitPath := filepath.Join(expandedSaveDir, activityID+".fit")
			if _, err := os.Stat(fitPath); err == nil {
				fmt.Printf("File already exists: %s\n", fitPath)
				continue
			}

			// Download FIT file
			fitData, _, err := client.GetFit(activityID)
			if err != nil {
				fmt.Printf("Failed to download FIT file for activity %s: %v\n", activityID, err)
				continue
			}

			// Save FIT file
			if err := os.WriteFile(fitPath, fitData, 0644); err != nil {
				fmt.Printf("Failed to save FIT file for activity %s: %v\n", activityID, err)
				continue
			}

			fmt.Printf("Saved FIT file: %s\n", fitPath)
			time.Sleep(1 * time.Second)
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

	// Add flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.runalyzedump.yaml)")
	rootCmd.Flags().StringVar(&username, "username", "", "Runalyze username")
	rootCmd.Flags().StringVar(&password, "password", "", "Runalyze password")
	rootCmd.Flags().StringVar(&cookiePath, "cookie-path", "", "Path to cookie file")
	rootCmd.PersistentFlags().String("save_dir", "~/proj/runalyzedump/output", "Directory to save downloaded files")

	// Bind flags to environment variables
	viper.BindEnv("username", "RUNALYZE_USERNAME")
	viper.BindEnv("password", "RUNALYZE_PASSWORD")
	viper.BindEnv("cookie_path", "RUNALYZE_COOKIE_PATH")
	viper.BindEnv("save_dir", "RUNALYZE_SAVE_DIR")
	viper.BindPFlag("save_dir", rootCmd.PersistentFlags().Lookup("save_dir"))
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

		// Search config in home directory with name ".runalyzedump" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".runalyzedump")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
