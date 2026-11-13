package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"proxy-service/internal/config"
	"proxy-service/internal/router"
)

// parseTarget validates a backend URL: it must parse and carry both a
// scheme and a host.
func parseTarget(target string) (*url.URL, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid backend URL %q: %w", target, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid backend URL %q: missing scheme or host", target)
	}
	return u, nil
}

// New builds a reverse proxy for the target URL.
func New(target string) *httputil.ReverseProxy {
	u, err := parseTarget(target)
	if err != nil {
		log.Fatal(err)
	}
	return httputil.NewSingleHostReverseProxy(u)
}

// Backends maps routing backends to their handlers (reverse proxies).
type Backends map[router.Backend]http.Handler

// NewBackends builds reverse proxies for every backend in the config.
func NewBackends(cfg config.Config) Backends {
	return Backends{
		router.Monolith: New(cfg.MonolithURL),
		router.Movies:   New(cfg.MoviesServiceURL),
		router.Events:   New(cfg.EventsServiceURL),
	}
}
