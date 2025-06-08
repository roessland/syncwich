package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/roessland/syncwich/runalyze"
	"github.com/roessland/syncwich/sw"
	"github.com/spf13/viper"
)

func main() {
	var (
		dryRun = flag.Bool("dry-run", false, "Show what would be updated without making changes")
	)
	flag.Parse()

	// Load credentials from config file or environment
	initConfig()
	username := viper.GetString("username")
	password := viper.GetString("password")

	if username == "" || password == "" {
		log.Fatal("SW_RUNALYZE_USERNAME and SW_RUNALYZE_PASSWORD required (via config file or environment)")
	}

	// Create temporary cookie file for this session
	tempCookiePath := filepath.Join(os.TempDir(), fmt.Sprintf("runalyze-fixtures-%d.json", time.Now().Unix()))
	defer os.Remove(tempCookiePath) // Clean up when done

	client, err := runalyze.New(username, password, tempCookiePath)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Login(); err != nil {
		log.Fatalf("Failed to login: %v", err)
	}

	fixturesDir := filepath.Join("sw", "testdata", "fixtures")
	if err := os.MkdirAll(fixturesDir, 0755); err != nil {
		log.Fatalf("Failed to create fixtures directory: %v", err)
	}

	// Find a week with 2+ activities in the last 4 weeks
	now := time.Now()

	// Start from last Monday
	weekStart := now
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	fmt.Println("Looking for a week with 2+ activities in the last 4 weeks...")

	var selectedWeek time.Time
	var selectedHTML []byte

	for i := 0; i < 4; i++ {
		currentWeek := weekStart.AddDate(0, 0, -7*i)
		fmt.Printf("Checking week %s...\n", currentWeek.Format("2006-01-02"))

		html, err := client.GetDataBrowser(currentWeek)
		if err != nil {
			log.Printf("Warning: Failed to fetch data for week %s: %v", currentWeek.Format("2006-01-02"), err)
			continue
		}

		// Count activities in this week
		activityCount := len(sw.FindActivityIds(html))
		fmt.Printf("  Found %d activities\n", activityCount)

		if activityCount >= 2 {
			selectedWeek = currentWeek
			selectedHTML = html
			fmt.Printf("✅ Selected week %s with %d activities\n", currentWeek.Format("2006-01-02"), activityCount)
			break
		}
	}

	if selectedWeek.IsZero() {
		log.Fatal("No week found with 2+ activities in the last 4 weeks. Please try again later or with different credentials.")
	}

	filename := fmt.Sprintf("%s-week.html", selectedWeek.Format("2006.01.02"))
	filepath := filepath.Join(fixturesDir, filename)

	if *dryRun {
		fmt.Printf("Would create: %s (%d bytes)\n", filepath, len(selectedHTML))
		return
	}

	if err := os.WriteFile(filepath, selectedHTML, 0644); err != nil {
		log.Fatalf("Failed to write fixture %s: %v", filepath, err)
	}

	fmt.Printf("✅ Created fixture: %s (%d bytes)\n", filepath, len(selectedHTML))
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Run tests: just test-fixtures\n")
	fmt.Printf("2. If first run, create golden files: just update-golden\n")
	fmt.Printf("3. Run tests again: just test-fixtures\n")
}

func initConfig() {
	// Check for config file
	home, err := homedir.Dir()
	if err != nil {
		log.Printf("Warning: Could not find home directory: %v", err)
		return
	}

	configPath := filepath.Join(home, ".syncwich", "syncwich.yaml")
	if _, err := os.Stat(configPath); err == nil {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Warning: Could not read config file: %v", err)
		}
	}

	// Also check environment variables
	viper.BindEnv("username", "SW_RUNALYZE_USERNAME")
	viper.BindEnv("password", "SW_RUNALYZE_PASSWORD")
}
