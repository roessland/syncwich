package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/roessland/syncwich/sw"
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
	Use:   "syncwich",
	Short: "A tool to sync and dump Runalyze data",
	Long: `Syncwich is a CLI tool to sync and dump Runalyze data for analysis and backup purposes.

It provides an interactive terminal interface with beautiful progress bars and structured logging.`,
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download activities from Runalyze",
	Long:  `Download activities from Runalyze with beautiful progress bars and structured logging.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		jsonMode, _ := cmd.Flags().GetBool("json")

		// Gather configuration from flags and viper
		config := sw.DownloadConfig{
			Username:   getConfigValue("", "username"),
			Password:   getConfigValue("", "password"),
			CookiePath: getConfigValue(cookiePath, "cookie_path"),
			UntilStr:   until,
			SinceStr:   since,
			SaveDir:    viper.GetString("save_dir"),
			JSONMode:   jsonMode,
		}

		// Call the business logic
		return sw.Download(config)
	},
}

// getConfigValue returns the flag value if non-empty, otherwise returns the viper config value
func getConfigValue(flagValue, viperKey string) string {
	if flagValue != "" {
		return flagValue
	}
	return viper.GetString(viperKey)
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

	// Viper defaults
	viper.SetDefault("save_dir", "~/.syncwich/activities")
	viper.SetDefault("cookie_path", "~/.syncwich/runalyze-cookie.json")

	// Here you will define your flags and configuration settings.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.syncwich/syncwich.yaml)")
	rootCmd.PersistentFlags().String("save_dir", "", "Directory to save downloaded files (default: ~/.syncwich/activities)")

	// Download command flags
	downloadCmd.Flags().String("since", "4w", "Download activities since this date (e.g., '2023-12-01', '30d', '4w')")
	downloadCmd.Flags().String("until", "", "Download activities until this date (optional)")
	downloadCmd.Flags().StringVar(&cookiePath, "cookie-path", "", "Path to cookie file (default: ~/.syncwich/runalyze-cookie.json)")
	downloadCmd.Flags().Bool("json", false, "Output structured JSON logs instead of interactive mode")

	// Bind environment variables
	viper.BindEnv("username", "SW_RUNALYZE_USERNAME")
	viper.BindEnv("password", "SW_RUNALYZE_PASSWORD")
	viper.BindEnv("cookie_path", "SW_RUNALYZE_COOKIE_PATH")
	viper.BindEnv("save_dir", "SW_RUNALYZE_SAVE_DIR")

	// Add download command to root
	rootCmd.AddCommand(downloadCmd)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in ~/.syncwich/ directory with name "syncwich" (without extension).
		viper.AddConfigPath(filepath.Join(home, ".syncwich"))
		viper.SetConfigName("syncwich")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in silently (logging is via LOG_LEVEL env var)
	viper.ReadInConfig()
}
