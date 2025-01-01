package middleware

import (
	"github.com/urfave/negroni"
	"net/http"
	"time"

	"go.uber.org/zap"
)

func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := negroni.NewResponseWriter(w)
		next.ServeHTTP(lrw, r)
		statusCode := lrw.Status()
		zap.S().Infow("New request",
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
			"url", r.URL.Path,
			"time", time.Since(start),
			"status", statusCode,
		)
	})
}
