package observability

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

type requestLogTiming struct {
	startedAt       time.Time
	executeStartedAt time.Time
	executeEndedAt   time.Time
	writeStartedAt   time.Time
	writeEndedAt     time.Time
}

type requestLogTimingKey struct{}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			startedAt := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			loggedReq := req.WithContext(WithRequestLogTiming(req.Context(), startedAt))
			next.ServeHTTP(recorder, loggedReq)
			attrs := []any{
				slog.String("method", loggedReq.Method),
				slog.String("path", loggedReq.URL.Path),
				slog.Int("status", recorder.statusCode),
				slog.Duration("duration", time.Since(startedAt)),
				slog.String("request_id", middleware.GetReqID(loggedReq.Context())),
			}
			if timing, ok := RequestLogTimingFromContext(loggedReq.Context()); ok {
				if !timing.executeStartedAt.IsZero() {
					attrs = append(attrs, slog.Duration("decode_and_preprocess_duration", timing.executeStartedAt.Sub(timing.startedAt)))
				}
				if !timing.executeStartedAt.IsZero() && !timing.executeEndedAt.IsZero() {
					attrs = append(attrs, slog.Duration("execute_gateway_duration", timing.executeEndedAt.Sub(timing.executeStartedAt)))
				}
				if !timing.writeStartedAt.IsZero() && !timing.writeEndedAt.IsZero() {
					attrs = append(attrs, slog.Duration("write_response_duration", timing.writeEndedAt.Sub(timing.writeStartedAt)))
				}
			}
			logger.Info("http_request", attrs...)
		})
	}
}

func WithRequestLogTiming(ctx context.Context, startedAt time.Time) context.Context {
	return context.WithValue(ctx, requestLogTimingKey{}, &requestLogTiming{startedAt: startedAt})
}

func RequestLogTimingFromContext(ctx context.Context) (*requestLogTiming, bool) {
	timing, ok := ctx.Value(requestLogTimingKey{}).(*requestLogTiming)
	return timing, ok && timing != nil
}

func MarkRequestExecuteStart(ctx context.Context) {
	if timing, ok := RequestLogTimingFromContext(ctx); ok {
		timing.executeStartedAt = time.Now()
	}
}

func MarkRequestExecuteEnd(ctx context.Context) {
	if timing, ok := RequestLogTimingFromContext(ctx); ok {
		timing.executeEndedAt = time.Now()
	}
}

func MarkRequestWriteStart(ctx context.Context) {
	if timing, ok := RequestLogTimingFromContext(ctx); ok {
		timing.writeStartedAt = time.Now()
	}
}

func MarkRequestWriteEnd(ctx context.Context) {
	if timing, ok := RequestLogTimingFromContext(ctx); ok {
		timing.writeEndedAt = time.Now()
	}
}
