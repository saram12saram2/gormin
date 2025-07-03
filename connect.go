package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

// GarminConnect handles communication with Garmin Connect
type GarminConnect struct {
	client   *http.Client
	username string
	password string
	baseURL  string
	loggedIn bool
}

// LoginResponse represents the login response from Garmin Connect
type LoginResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ActivityListResponse represents the activity list response
type ActivityListResponse struct {
	Activities []Activity `json:"activities"`
}

// GarminActivity represents an activity from Garmin Connect API
type GarminActivity struct {
	ActivityID      int     `json:"activityId"`
	ActivityName    string  `json:"activityName"`
	ActivityTypeKey string  `json:"activityTypeKey"`
	StartTimeLocal  string  `json:"startTimeLocal"`
	Duration        float64 `json:"duration"`
	Distance        float64 `json:"distance"`
	Calories        float64 `json:"calories"`
	AverageHR       float64 `json:"averageHR"`
	MaxHR           float64 `json:"maxHR"`
	ElevationGain   float64 `json:"elevationGain"`
}

// GarminDailyStats represents daily statistics from Garmin Connect
type GarminDailyStats struct {
	CalendarDate   string  `json:"calendarDate"`
	TotalSteps     int     `json:"totalSteps"`
	TotalDistance  float64 `json:"totalDistance"`
	ActiveCalories int     `json:"activeCalories"`
	RestingHR      int     `json:"restingHeartRate"`
	SleepDuration  int     `json:"sleepDuration"`
	BodyWeight     float64 `json:"bodyWeight"`
	BodyFatPercent float64 `json:"bodyFatPercent"`
}

// NewGarminConnect creates a new Garmin Connect client
func NewGarminConnect(username, password string) *GarminConnect {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	return &GarminConnect{
		client:   client,
		username: username,
		password: password,
		baseURL:  "https://connect.garmin.com",
		loggedIn: false,
	}
}

// Login authenticates with Garmin Connect
func (gc *GarminConnect) Login() error {
	fmt.Println("Logging into Garmin Connect...")

	// Step 1: Get SSO login page
	ssoURL := "https://sso.garmin.com/sso/signin"
	params := url.Values{}
	params.Set("service", "https://connect.garmin.com/modern/")
	params.Set("webhost", "https://connect.garmin.com/modern/")
	params.Set("source", "https://connect.garmin.com/signin/")
	params.Set("redirectAfterAccountLoginUrl", "https://connect.garmin.com/modern/")
	params.Set("redirectAfterAccountCreationUrl", "https://connect.garmin.com/modern/")
	params.Set("gauthHost", "https://sso.garmin.com/sso")
	params.Set("locale", "en_US")
	params.Set("id", "gauth-widget")
	params.Set("cssUrl", "https://static.garmincdn.com/com.garmin.connect/ui/css/gauth-custom-v1.2-min.css")
	params.Set("privacyStatementUrl", "https://www.garmin.com/en-US/privacy/connect/")
	params.Set("clientId", "GarminConnect")
	params.Set("rememberMeShown", "true")
	params.Set("rememberMeChecked", "false")
	params.Set("createAccountShown", "true")
	params.Set("openCreateAccount", "false")
	params.Set("displayNameShown", "false")
	params.Set("consumeServiceTicket", "false")
	params.Set("initialFocus", "true")
	params.Set("embedWidget", "false")
	params.Set("generateExtraServiceTicket", "true")
	params.Set("generateTwoExtraServiceTickets", "false")
	params.Set("generateNoServiceTicket", "false")
	params.Set("globalOptInShown", "true")
	params.Set("globalOptInChecked", "false")
	params.Set("mobile", "false")
	params.Set("connectLegalTerms", "true")
	params.Set("showTermsOfUse", "false")
	params.Set("showPrivacyPolicy", "false")
	params.Set("showConnectLegalAge", "false")
	params.Set("locationPromptShown", "true")
	params.Set("showPassword", "true")
	params.Set("useCustomHeader", "false")

	loginPageURL := ssoURL + "?" + params.Encode()

	resp, err := gc.client.Get(loginPageURL)
	if err != nil {
		return fmt.Errorf("failed to get login page: %w", err)
	}
	defer resp.Body.Close()

	// Step 2: Submit login credentials
	loginData := url.Values{}
	loginData.Set("username", gc.username)
	loginData.Set("password", gc.password)
	loginData.Set("embed", "false")
	loginData.Set("_eventId", "submit")
	loginData.Set("displayNameRequired", "false")

	req, err := http.NewRequest("POST", ssoURL, strings.NewReader(loginData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "GarminDB-Go/1.0")
	req.Header.Set("Referer", loginPageURL)

	resp, err = gc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to submit login: %w", err)
	}
	defer resp.Body.Close()

	// Check if login was successful by looking for service tickets
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	responseBody := string(body)
	if strings.Contains(responseBody, "Invalid username or password") {
		return fmt.Errorf("invalid username or password")
	}

	// Extract service ticket from response
	if !strings.Contains(responseBody, "ticket=") {
		return fmt.Errorf("login failed: no service ticket found")
	}

	gc.loggedIn = true
	fmt.Println("Successfully logged into Garmin Connect")
	return nil
}

// GetActivities retrieves activities from Garmin Connect
func (gc *GarminConnect) GetActivities(limit, start int) ([]Activity, error) {
	if !gc.loggedIn {
		if err := gc.Login(); err != nil {
			return nil, err
		}
	}

	fmt.Printf("Fetching %d activities starting from %d...\n", limit, start)

	url := fmt.Sprintf("%s/modern/proxy/activitylist-service/activities/search/activities?limit=%d&start=%d",
		gc.baseURL, limit, start)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "GarminDB-Go/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("NK", "NT") // Required for some Garmin Connect endpoints

	resp, err := gc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get activities: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var garminActivities []GarminActivity
	if err := json.Unmarshal(body, &garminActivities); err != nil {
		return nil, fmt.Errorf("failed to parse activities: %w", err)
	}

	// Convert to our Activity struct
	var activities []Activity
	for _, ga := range garminActivities {
		activity := Activity{
			ID:            ga.ActivityID,
			Name:          ga.ActivityName,
			Type:          ga.ActivityTypeKey,
			StartTime:     ga.StartTimeLocal,
			Duration:      int(ga.Duration),
			Distance:      ga.Distance / 1000.0, // Convert meters to km
			Calories:      int(ga.Calories),
			AvgHR:         int(ga.AverageHR),
			MaxHR:         int(ga.MaxHR),
			ElevationGain: int(ga.ElevationGain),
		}
		activities = append(activities, activity)
	}

	fmt.Printf("Retrieved %d activities\n", len(activities))
	return activities, nil
}

// GetDailyStats retrieves daily statistics
func (gc *GarminConnect) GetDailyStats(date time.Time) (*DailyStats, error) {
	if !gc.loggedIn {
		if err := gc.Login(); err != nil {
			return nil, err
		}
	}

	dateStr := date.Format("2006-01-02")
	url := fmt.Sprintf("%s/modern/proxy/userstats-service/wellness/daily/%s",
		gc.baseURL, dateStr)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "GarminDB-Go/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("NK", "NT")

	resp, err := gc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get daily stats: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var garminStats GarminDailyStats
	if err := json.Unmarshal(body, &garminStats); err != nil {
		return nil, fmt.Errorf("failed to parse daily stats: %w", err)
	}

	// Convert to our DailyStats struct
	stats := &DailyStats{
		Date:       garminStats.CalendarDate,
		Steps:      garminStats.TotalSteps,
		Distance:   garminStats.TotalDistance / 1000.0, // Convert meters to km
		Calories:   garminStats.ActiveCalories,
		SleepHours: float64(garminStats.SleepDuration) / 3600.0, // Convert seconds to hours
		RestingHR:  garminStats.RestingHR,
		Weight:     garminStats.BodyWeight,
		BodyFat:    garminStats.BodyFatPercent,
	}

	return stats, nil
}

// DownloadFitFile downloads a FIT file for an activity
func (gc *GarminConnect) DownloadFitFile(activityID int, outputPath string) error {
	if !gc.loggedIn {
		if err := gc.Login(); err != nil {
			return err
		}
	}

	url := fmt.Sprintf("%s/modern/proxy/download-service/files/activity/%d",
		gc.baseURL, activityID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "GarminDB-Go/1.0")
	req.Header.Set("NK", "NT")

	resp, err := gc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download FIT file: status %d", resp.StatusCode)
	}

	// Write to file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
