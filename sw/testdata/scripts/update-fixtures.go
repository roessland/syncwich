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
		home, _ := homedir.Dir()
		configPath := filepath.Join(home, ".syncwich", "syncwich.yaml")
		log.Fatalf(`Credentials not found. Either:

  Config file at %s:
    username: your_username
    password: your_password

  Or environment variables:
    export SW_RUNALYZE_USERNAME=your_username
    export SW_RUNALYZE_PASSWORD=your_password
`, configPath)
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

	var selectedWeek time.Time
	var selectedHTML []byte

	// Check for existing fixture - reuse the same week if possible
	existingFixtures, _ := filepath.Glob(filepath.Join(fixturesDir, "*.html"))
	if len(existingFixtures) > 0 {
		// Parse week from existing fixture filename (e.g., "2025.05.26-week.html")
		basename := filepath.Base(existingFixtures[0])
		dateStr := basename[:10] // "2025.05.26"
		if parsed, err := time.Parse("2006.01.02", dateStr); err == nil {
			fmt.Printf("Refreshing existing fixture for week %s...\n", parsed.Format("2006-01-02"))
			html, err := client.GetDataBrowser(parsed)
			if err != nil {
				log.Fatalf("Failed to fetch data for week %s: %v", parsed.Format("2006-01-02"), err)
			}
			selectedWeek = parsed
			selectedHTML = html
		}
	}

	// If no existing fixture, find a week with 2+ activities
	if selectedWeek.IsZero() {
		now := time.Now()

		// Start from last Monday
		weekStart := now
		for weekStart.Weekday() != time.Monday {
			weekStart = weekStart.AddDate(0, 0, -1)
		}
		weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

		fmt.Println("Looking for a week with 2+ activities in the last 4 weeks...")

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
			log.Fatal("No week found with 2+ activities in the last 4 weeks.")
		}
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

	// Check if golden file exists
	goldenPath := fmt.Sprintf("sw/testdata/golden/%s-week.json", selectedWeek.Format("2006.01.02"))
	if _, err := os.Stat(goldenPath); err == nil {
		fmt.Printf("\nNext: run 'just test-fixtures'\n")
	} else {
		fmt.Printf("\nNext: run 'just update-golden' then 'just test-fixtures'\n")
	}
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
