package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/SatarYaghoubi/sentinel/internal/scanner"
	"github.com/SatarYaghoubi/sentinel/internal/store"
)

// maxUploadBytes caps the size of an accepted upload (10 MiB).
const maxUploadBytes = 10 << 20

// Handler holds the dependencies shared across HTTP handlers.
type Handler struct {
	Scanner *scanner.Scanner
	Store   *store.Store
	started time.Time
}

// NewHandler builds a Handler.
func NewHandler(sc *scanner.Scanner, st *store.Store) *Handler {
	return &Handler{Scanner: sc, Store: st, started: time.Now()}
}

// Health reports basic liveness and uptime information.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"uptime":       time.Since(h.started).String(),
		"scans_stored": h.Store.Count(),
	})
}

// Scan accepts a multipart file upload, scans it, stores the result and
// returns the report.
func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "could not parse upload (is it multipart and under 10 MiB?)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' field in multipart form")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read uploaded file")
		return
	}

	result, err := h.Scanner.Scan(r.Context(), data)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "scan could not be completed")
		return
	}

	rec := store.Record{
		ID:       newID(),
		Filename: header.Filename,
		Result:   result,
	}
	h.Store.Save(rec)

	status := http.StatusOK
	if !result.Clean {
		status = http.StatusUnprocessableEntity // flagged content
	}
	writeJSON(w, status, rec)
}

// GetResult returns a previously stored scan record by ID.
func (h *Handler) GetResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rec, ok := h.Store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "no scan found with that id")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

// ListResults returns all stored scan records.
func (h *Handler) ListResults(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"count":   h.Store.Count(),
		"results": h.Store.List(),
	})
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// newID returns a short random hex identifier for a scan record.
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
