package api

import (
	"net/http"

	"github.com/SatarYaghoubi/sentinel/internal/ratelimit"
)

// NewRouter wires routes and middleware together and returns the root handler.
func NewRouter(h *Handler, limiter *ratelimit.Limiter) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("POST /v1/scan", h.Scan)
	mux.HandleFunc("GET /v1/scans", h.ListResults)
	mux.HandleFunc("GET /v1/scans/{id}", h.GetResult)

	// Order: recover (outermost) -> log -> rate limit -> mux.
	return chain(mux,
		Recover,
		Logging,
		RateLimit(limiter),
	)
}
