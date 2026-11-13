package router

import "testing"

func TestChooseBackendBoundaries(t *testing.T) {
	// The edges are deterministic and do not depend on the random roll
	for i := 0; i < 100; i++ {
		if ChooseBackend(0) {
			t.Fatal("ChooseBackend(0) = true, want false")
		}
		if ChooseBackend(-10) {
			t.Fatal("ChooseBackend(-10) = true, want false")
		}
		if !ChooseBackend(100) {
			t.Fatal("ChooseBackend(100) = false, want true")
		}
		if !ChooseBackend(150) {
			t.Fatal("ChooseBackend(150) = false, want true")
		}
	}
}

func TestChooseBackendDistribution(t *testing.T) {
	// At 50% a large number of rolls should hit both backends with a
	// roughly even split
	movies := 0
	total := 1000
	for i := 0; i < total; i++ {
		if ChooseBackend(50) {
			movies++
		}
	}
	if movies == 0 || movies == total {
		t.Fatalf("ChooseBackend at 50%% routed %d/%d rolls to movies, expected a mix", movies, total)
	}
	if movies < 350 || movies > 650 {
		t.Errorf("ChooseBackend at 50%% routed %d/%d rolls to movies, want within 35-65%%", movies, total)
	}
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		gradual bool
		percent int
		want    Backend
	}{
		{"movies with gradual off goes to monolith", "/api/movies", false, 100, Monolith},
		{"movies at 100 percent goes to movies service", "/api/movies", true, 100, Movies},
		{"movies at 0 percent goes to monolith", "/api/movies", true, 0, Monolith},
		{"events goes to events service", "/api/events", false, 0, Events},
		{"users goes to monolith", "/api/users", true, 100, Monolith},
		{"unknown path goes to monolith", "/something/else", true, 100, Monolith},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Decide(tt.path, tt.gradual, tt.percent); got != tt.want {
				t.Errorf("Decide(%q, %t, %d) = %q, want %q",
					tt.path, tt.gradual, tt.percent, got, tt.want)
			}
		})
	}
}

func TestDecideMoviesSplit(t *testing.T) {
	// With gradual migration at 50% both backends should show up
	seen := map[Backend]bool{}
	for i := 0; i < 100; i++ {
		seen[Decide("/api/movies", true, 50)] = true
	}
	if !seen[Monolith] || !seen[Movies] {
		t.Errorf("Decide at 50%% saw only %v, expected both monolith and movies-service", seen)
	}
}
