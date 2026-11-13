package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"proxy-service/internal/config"
	"proxy-service/internal/proxy"
	"proxy-service/internal/router"
)

// Gateway is the HTTP entry point of the proxy service: it routes requests
// to backends according to the migration configuration.
type Gateway struct {
	cfg      config.Config
	backends proxy.Backends
}

// New creates a Gateway serving requests over the given backends.
func New(cfg config.Config, backends proxy.Backends) *Gateway {
	return &Gateway{cfg: cfg, backends: backends}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		handleHealth(w, r)
		return
	}

	backend := router.Decide(r.URL.Path, g.cfg.GradualMigration, g.cfg.MoviesMigrationPct)

	if strings.HasPrefix(r.URL.Path, "/api/movies") {
		if g.cfg.GradualMigration {
			log.Printf("Routing %s %s (percent=%d) -> %s", r.Method, r.URL.Path, g.cfg.MoviesMigrationPct, backend)
		} else {
			log.Printf("Routing %s %s (gradual migration off) -> %s", r.Method, r.URL.Path, backend)
		}
	} else {
		log.Printf("Routing %s %s -> %s", r.Method, r.URL.Path, backend)
	}

	g.backends[backend].ServeHTTP(w, r)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}
