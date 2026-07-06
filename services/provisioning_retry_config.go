package services

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func provisioningWorkerInterval() time.Duration {
	return durationEnv("PROVISIONING_RETRY_INTERVAL", time.Minute)
}

func provisioningMaxAttempts() int {
	return intEnv("PROVISIONING_MAX_ATTEMPTS", 3)
}

func provisioningRetryBatchSize() int {
	return intEnv("PROVISIONING_RETRY_BATCH_SIZE", 25)
}

func provisioningRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	base := durationEnv("PROVISIONING_RETRY_BASE_DELAY", time.Minute)
	maxDelay := durationEnv("PROVISIONING_RETRY_MAX_DELAY", 30*time.Minute)
	delay := base * time.Duration(attempt)
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err == nil && duration > 0 {
		return duration
	}
	seconds, err := strconv.Atoi(value)
	if err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
