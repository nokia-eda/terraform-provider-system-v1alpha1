package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

const ENV_LOG_LEVEL = "LOG_LEVEL"

func GetLogLevel() slog.Level {
	logLevel := GetEnvWithDefault(ENV_LOG_LEVEL, "info")
	switch logLevel {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func GetEnv(key string) (string, error) {
	if val := os.Getenv(key); val != "" {
		return val, nil
	}
	return "", fmt.Errorf("unable to find environment variable: %s", key)
}

func GetEnvWithDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func GetEnvBoolWithDefault(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		bVal, err := strconv.ParseBool(val)
		if err != nil {
			return defaultVal
		}
		return bVal
	}
	return defaultVal
}

func GetEnvIntWithDefault(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if ret, err := strconv.Atoi(val); err == nil {
			return ret
		}
	}
	return defaultVal
}

func GetEnvDurationWithDefault(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if dur, err := time.ParseDuration(val); err == nil {
			return dur
		}
	}
	return defaultVal
}

func ToJSON(input any) (string, error) {
	bytes, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func Convert[T any](value any, target *T) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, target)
}
