package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DB struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Database string `json:"database"`
	} `json:"database"`
	RabbitMQ struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
	} `json:"rabbitmq"`
	Websocket struct {
		Port int `json:"port"`
	} `json:"websocket"`
	Services struct {
		RideService           int `json:"ride_service"`
		DriverLocationService int `json:"driver_location_service"`
		AdminService          int `json:"admin_service"`
	} `json:"services"`
}

func loadEnvFile() error {
	file, err := os.Open(".env")
	if err != nil {
		return fmt.Errorf("could not open env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Trim spaces and ignore comments or empty lines
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split into key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // or return error if strict
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove optional surrounding quotes
		value = strings.Trim(value, `"'`)

		err := os.Setenv(key, value)
		if err != nil {
			return fmt.Errorf("could not set env var %s: %w", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading env file: %w", err)
	}

	return nil
}

func LoadConfig() (*Config, error) {
	err := loadEnvFile()
	if err != nil {
		return nil, err
	}
	b, err := yAMLToJSON()
	if err != nil {
		return nil, err
	}
	cfg := new(Config)
	err = json.Unmarshal(b, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func yAMLToJSON() ([]byte, error) {
	file, err := os.Open("config.yml")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make(map[string]map[string]any)
	var currentSection string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section (top-level key ending with ":")
		if strings.HasSuffix(line, ":") {
			currentSection = strings.TrimSuffix(line, ":")
			result[currentSection] = make(map[string]any)
			continue
		}

		// Parse key: value lines
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := customGetEnv(strings.TrimSpace(parts[1]))
		if val == "" {
			return nil, fmt.Errorf("not seted val to key: %s", key)
		}
		// Try to convert to int if possible
		var v any = val
		var intVal int
		if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
			v = intVal
		} else {
			// remove quotes if exist
			v = strings.Trim(val, `"'`)
		}

		if currentSection != "" {
			result[currentSection][key] = v
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Marshal map to JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func customGetEnv(str string) string {
	// Проверяем, начинается ли строка с "${" и заканчивается ли "}"
	if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") {
		// Убираем "${" и "}"
		inside := str[2 : len(str)-1]

		// Разделяем по ":-" на переменную и дефолтное значение
		parts := strings.SplitN(inside, ":-", 2)
		envVar := parts[0]
		defValue := ""
		if len(parts) == 2 {
			defValue = parts[1]
		}

		// Получаем значение из окружения
		val, exists := os.LookupEnv(envVar)
		if exists {
			return val
		}
		return defValue
	}
	return str
}
