package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
)

type Config struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	ActivityID int    `json:"activity_id"`
	Output     string `json:"output"`
}

func main() {
	config := loadConfig("config.json")

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	err := loginGarmin(client, config.Username, config.Password)
	if err != nil {
		fmt.Printf("Ошибка login: %v\n", err)
		return
	}

	fmt.Println("Скачиваем .fit файл...")
	err = downloadFitFile(client, config.ActivityID, config.Output)
	if err != nil {
		fmt.Printf("Ошибка при загрузке файла: %v\n", err)
		return
	}
	fmt.Printf("Файл успешно загружен: %s\n", config.Output)
}

func loadConfig(filename string) Config {
	file, err := os.Open(filename)
	if err != nil {
		os.Exit(1)
	}
	defer file.Close()

	var cfg Config
	err = json.NewDecoder(file).Decode(&cfg)
	if err != nil {
		fmt.Printf("[loadConfig] Ошибка чтения JSON: %v\n", err)
		os.Exit(1)
	}
	// fmt.Printf("[loadConfig] Конфигурация загружена: %+v\n", cfg)
	return cfg
}

func loginGarmin(client *http.Client, username, password string) error {
	data := fmt.Sprintf("username=%s&password=%s", username, password)
	req, _ := http.NewRequest("POST", "https://connect.garmin.com/signin", strings.NewReader(data))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "SimpleGarminClient/1.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[loginGarmin] Ошибка HTTP-запроса: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Логин не удался. Код статуса: %d\n", resp.StatusCode)
		return fmt.Errorf("status: %d", resp.StatusCode)
	}
	return nil
}

func downloadFitFile(client *http.Client, activityID int, filename string) error {
	fmt.Printf("[downloadFitFile] Скачиваем файл активности под ID=%d...\n", activityID)

	url := fmt.Sprintf("https://connect.garmin.com/modern/proxy/download-service/files/activity/%d", activityID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "SimpleGarminClient/1.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[downloadFitFile] Ошибка HTTP-запроса: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("[downloadFitFile] Ошибка: получен статус %d\n", resp.StatusCode)
		return fmt.Errorf("status: %d", resp.StatusCode)
	}

	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("[downloadFitFile] Не удалось создать файл: %v\n", err)
		return err
	}
	defer file.Close()

	fmt.Printf("[downloadFitFile] Сохраняем данные в файл %s...\n", filename)
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("[downloadFitFile] Ошибка при записи файла: %v\n", err)
		return err
	}

	return nil
}
