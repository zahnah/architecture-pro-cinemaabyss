package config

import "testing"

func TestClampPercent(t *testing.T) {
	tests := []struct {
		raw  string
		want int
	}{
		{"0", 0},
		{"50", 50},
		{"100", 100},
		{"-5", 0},
		{"250", 100},
		{"not-a-number", 0},
		{"", 0},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := clampPercent(tt.raw); got != tt.want {
				t.Errorf("clampPercent(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	// Empty values exercise the fallback path of getEnv
	t.Setenv("PORT", "")
	t.Setenv("MONOLITH_URL", "")
	t.Setenv("MOVIES_SERVICE_URL", "")
	t.Setenv("EVENTS_SERVICE_URL", "")
	t.Setenv("GRADUAL_MIGRATION", "")
	t.Setenv("MOVIES_MIGRATION_PERCENT", "")

	cfg := Load()

	if cfg.Port != "8000" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8000")
	}
	if cfg.MonolithURL != "http://localhost:8080" {
		t.Errorf("MonolithURL = %q, want %q", cfg.MonolithURL, "http://localhost:8080")
	}
	if cfg.MoviesServiceURL != "http://localhost:8081" {
		t.Errorf("MoviesServiceURL = %q, want %q", cfg.MoviesServiceURL, "http://localhost:8081")
	}
	if cfg.EventsServiceURL != "http://localhost:8082" {
		t.Errorf("EventsServiceURL = %q, want %q", cfg.EventsServiceURL, "http://localhost:8082")
	}
	if cfg.GradualMigration != false {
		t.Errorf("GradualMigration = %t, want false", cfg.GradualMigration)
	}
	if cfg.MoviesMigrationPct != 0 {
		t.Errorf("MoviesMigrationPct = %d, want 0", cfg.MoviesMigrationPct)
	}
}
