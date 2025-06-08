package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/roessland/runalyzedump/rd"
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
		// Gather configuration from flags and viper
		config := rd.DownloadConfig{
			Username:   getConfigValue(username, "username"),
			Password:   getConfigValue(password, "password"),
			CookiePath: getConfigValue(cookiePath, "cookie_path"),
			UntilStr:   untilStr,
			SinceStr:   sinceStr,
			SaveDir:    viper.GetString("save_dir"),
			JSONMode:   jsonMode,
		}

		// Call the business logic
		return rd.Download(config)
	},
}

// getConfigValue returns the flag value if set, otherwise falls back to viper config
func getConfigValue(flagValue, configKey string) string {
	if flagValue != "" {
		return flagValue
	}
	return viper.GetString(configKey)
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
