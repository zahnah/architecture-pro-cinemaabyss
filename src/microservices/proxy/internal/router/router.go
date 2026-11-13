package router

import (
	"math/rand/v2"
	"strings"
)

// Backend identifies an upstream service. The string values are used in
// routing logs.
type Backend string

// Backend names used in routing logs
const (
	Monolith Backend = "monolith"
	Movies   Backend = "movies-service"
	Events   Backend = "events-service"
)

// ChooseBackend decides randomly per request whether a /api/movies request
// goes to the movies microservice (true) or to the monolith (false).
// The edges are deterministic: percent <= 0 is always false, percent >= 100
// is always true.
func ChooseBackend(percent int) bool {
	if percent <= 0 {
		return false
	}
	if percent >= 100 {
		return true
	}
	return rand.IntN(100) < percent
}

// Decide picks the backend for the request path. /api/movies traffic is
// split between the monolith and the movies service when gradual migration
// is enabled; /api/events always goes to the events service; everything else
// goes to the monolith.
func Decide(path string, gradual bool, percent int) Backend {
	switch {
	case strings.HasPrefix(path, "/api/movies"):
		if gradual && ChooseBackend(percent) {
			return Movies
		}
		return Monolith
	case strings.HasPrefix(path, "/api/events"):
		return Events
	default:
		return Monolith
	}
}
