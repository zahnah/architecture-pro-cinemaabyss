package config

import (
	"log"
	"os"
	"strconv"
)

// Config holds the proxy-service configuration loaded from environment.
type Config struct {
	Port               string
	MonolithURL        string
	MoviesServiceURL   string
	EventsServiceURL   string
	GradualMigration   bool
	MoviesMigrationPct int
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Load reads configuration from environment variables, falling back to
// defaults and logging about invalid values.
func Load() Config {
	gradualMigration, err := strconv.ParseBool(getEnv("GRADUAL_MIGRATION", "false"))
	if err != nil {
		log.Printf("Invalid GRADUAL_MIGRATION value, defaulting to false: %v", err)
		gradualMigration = false
	}

	return Config{
		Port:               getEnv("PORT", "8000"),
		MonolithURL:        getEnv("MONOLITH_URL", "http://localhost:8080"),
		MoviesServiceURL:   getEnv("MOVIES_SERVICE_URL", "http://localhost:8081"),
		EventsServiceURL:   getEnv("EVENTS_SERVICE_URL", "http://localhost:8082"),
		GradualMigration:   gradualMigration,
		MoviesMigrationPct: clampPercent(getEnv("MOVIES_MIGRATION_PERCENT", "0")),
	}
}

// clampPercent parses percent and clamps it into the [0, 100] range
func clampPercent(raw string) int {
	p, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("Invalid MOVIES_MIGRATION_PERCENT value %q, defaulting to 0", raw)
		return 0
	}
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}
