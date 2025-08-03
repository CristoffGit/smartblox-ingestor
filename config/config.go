package config

import "os"

// Reads an environment variable or returns a fallback value.
func GetEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
