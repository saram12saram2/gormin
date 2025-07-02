package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	DatabasePath   string `json:"database_path"`
	DataPath       string `json:"data_path"`
	GarminUsername string `json:"garmin_username"`
	GarminPassword string `json:"garmin_password"`
	RetainFiles    bool   `json:"retain_files"`
	DownloadDays   int    `json:"download_days"`
}

var (
	config *Config
	db     *sql.DB
)

// fitness activity
type Activity struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	StartTime     string  `json:"start_time"`
	Duration      int     `json:"duration"`
	Distance      float64 `json:"distance"`
	Calories      int     `json:"calories"`
	AvgHR         int     `json:"avg_hr"`
	MaxHR         int     `json:"max_hr"`
	ElevationGain int     `json:"elevation_gain"`
}

// daily health statistics
type DailyStats struct {
	Date       string  `json:"date"`
	Steps      int     `json:"steps"`
	Distance   float64 `json:"distance"`
	Calories   int     `json:"calories"`
	SleepHours float64 `json:"sleep_hours"`
	RestingHR  int     `json:"resting_hr"`
	Weight     float64 `json:"weight"`
	BodyFat    float64 `json:"body_fat"`
}

func main() {
	var configPath = flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	// Load configuration
	if err := loadConfig(*configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	if err := initDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Parse command
	if len(flag.Args()) == 0 {
		return
	}

	command := flag.Args()[0]
	switch command {
	case "init":
		fmt.Println("Database initialized successfully")
	case "version":
		fmt.Println("GarminDB Go v1.0.0")
	default:
		fmt.Printf("Unknown command: %s\n", command)
	}
}

func loadConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	config = &Config{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(config.DataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	return nil
}

// initDatabase initializes the SQLite database and creates tables
func initDatabase() error {
	var err error
	db, err = sql.Open("sqlite3", config.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// createTables creates all necessary database tables
func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS activities (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			duration INTEGER,
			distance REAL,
			calories INTEGER,
			avg_hr INTEGER,
			max_hr INTEGER,
			elevation_gain INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS daily_stats (
			date TEXT PRIMARY KEY,
			steps INTEGER,
			distance REAL,
			calories INTEGER,
			sleep_hours REAL,
			resting_hr INTEGER,
			weight REAL,
			body_fat REAL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS weight_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			weight REAL NOT NULL,
			body_fat REAL,
			muscle_mass REAL,
			bone_mass REAL,
			water_percentage REAL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS heart_rate (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			heart_rate INTEGER NOT NULL,
			activity_id INTEGER,
			FOREIGN KEY (activity_id) REFERENCES activities (id)
		)`,

		`CREATE TABLE IF NOT EXISTS sleep_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			start_time DATETIME,
			end_time DATETIME,
			duration INTEGER,
			deep_sleep INTEGER,
			light_sleep INTEGER,
			rem_sleep INTEGER,
			awake_time INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	// Create indexes for better performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_activities_start_time ON activities(start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_activities_type ON activities(type)`,
		`CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats(date)`,
		`CREATE INDEX IF NOT EXISTS idx_weight_data_date ON weight_data(date)`,
		`CREATE INDEX IF NOT EXISTS idx_heart_rate_timestamp ON heart_rate(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_sleep_data_date ON sleep_data(date)`,
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
