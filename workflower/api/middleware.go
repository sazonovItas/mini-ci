package api

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/containerd/log"
)

func (a *API) withMiddleware(h http.Handler) http.Handler {
	return a.cors(a.logger(a.recoverer(h)))
}

func (a *API) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				log.G(r.Context()).Errorf("PANIC: %+v\n%s", rvr, debug.Stack())
				respondErrorMessage(w, http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %+v", rvr))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (a *API) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.G(r.Context()).Infof(
			"HTTP %s %s %d %s",
			r.Method,
			r.URL.Path,
			rw.status,
			duration,
		)
	})
}

func (a *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wrote {
		return
	}
	rw.status = code
	rw.wrote = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wrote {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
