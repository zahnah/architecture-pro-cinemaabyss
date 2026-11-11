package main

import (
	"log"
	"net/http"

	"proxy-service/internal/config"
	"proxy-service/internal/handler"
	"proxy-service/internal/proxy"
)

func main() {
	cfg := config.Load()
	backends := proxy.NewBackends(cfg)
	gateway := handler.New(cfg, backends)

	log.Printf("Starting proxy service on port %s (gradual migration: %t, movies migration percent: %d)",
		cfg.Port, cfg.GradualMigration, cfg.MoviesMigrationPct)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, gateway))
}
