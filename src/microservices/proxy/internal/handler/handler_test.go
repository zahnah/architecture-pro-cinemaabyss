package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"proxy-service/internal/config"
	"proxy-service/internal/proxy"
	"proxy-service/internal/router"
)

// stubBackends returns backends that record which one handled the request
// in the X-Backend response header.
func stubBackends() proxy.Backends {
	stub := func(b router.Backend) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Backend", string(b))
			w.WriteHeader(http.StatusOK)
		})
	}
	return proxy.Backends{
		router.Monolith: stub(router.Monolith),
		router.Movies:   stub(router.Movies),
		router.Events:   stub(router.Events),
	}
}

func newGateway(gradual bool, percent int) *Gateway {
	cfg := config.Config{
		GradualMigration:   gradual,
		MoviesMigrationPct: percent,
	}
	return New(cfg, stubBackends())
}

func TestHealthNotRoutedToBackends(t *testing.T) {
	g := newGateway(true, 100)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	g.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/health status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body != "{\"status\":true}\n" {
		t.Errorf("/health body = %q, want %q", body, "{\"status\":true}\n")
	}
	if backend := rec.Header().Get("X-Backend"); backend != "" {
		t.Errorf("/health was routed to backend %q, want no backend", backend)
	}
}

func TestRoutesToBackends(t *testing.T) {
	tests := []struct {
		name    string
		gradual bool
		percent int
		method  string
		path    string
		headers map[string]string
		want    router.Backend
	}{
		{"users path goes to monolith", true, 100, http.MethodGet, "/api/users", nil, router.Monolith},
		{"movies with gradual off goes to monolith", false, 100, http.MethodGet, "/api/movies", nil, router.Monolith},
		{"movies at 100 percent goes to movies service", true, 100, http.MethodGet, "/api/movies", nil, router.Movies},
		{"movies at 0 percent goes to monolith", true, 0, http.MethodGet, "/api/movies", nil, router.Monolith},
		{"events goes to events service", false, 0, http.MethodGet, "/api/events", nil, router.Events},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newGateway(tt.gradual, tt.percent)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			g.ServeHTTP(rec, req)

			if got := rec.Header().Get("X-Backend"); got != string(tt.want) {
				t.Errorf("%s %s routed to %q, want %q", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestMoviesRandomSplit(t *testing.T) {
	// With gradual migration at 50% both backends should show up over
	// repeated requests
	g := newGateway(true, 50)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/movies", nil)
		g.ServeHTTP(rec, req)
		seen[rec.Header().Get("X-Backend")] = true
	}
	if !seen[string(router.Monolith)] || !seen[string(router.Movies)] {
		t.Errorf("random split at 50%% saw only %v, expected both backends", seen)
	}
}
