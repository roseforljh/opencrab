package httpserver

import (
	"net/http"
	"time"
)

func healthHandler(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    status,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}
