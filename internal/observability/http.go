package observability

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			startedAt := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(recorder, req)

			logger.Info("http_request",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", recorder.statusCode),
				slog.Duration("duration", time.Since(startedAt)),
				slog.String("request_id", middleware.GetReqID(req.Context())),
			)
		})
	}
}
