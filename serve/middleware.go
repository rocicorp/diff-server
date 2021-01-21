package serve

import (
	"context"
	"net/http"
	"sync/atomic"

	zl "github.com/rs/zerolog"
	"roci.dev/diff-server/util/log"
	"roci.dev/diff-server/util/loghttp"
)

// contectLogger is http middleware that inserts a contextual logger into
// the http.Request's Context.
func contextLogger(next http.Handler) http.Handler {
	return &contextLoggerHandler{next: next}
}

type contextLoggerHandler struct {
	next  http.Handler
	reqID uint64
}

func (c *contextLoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lc := log.Default().With().Str("req", r.URL.String()).Uint64("rid", atomic.AddUint64(&c.reqID, 1))
	syncID := r.Header.Get("X-Replicache-SyncID")
	if syncID != "" {
		lc = lc.Str("syncID", syncID)
	}
	l := lc.Logger()
	ctx := context.WithValue(r.Context(), loggerKey{}, l)
	r = r.WithContext(ctx)
	c.next.ServeHTTP(w, r)
}

type loggerKey struct{}

func logger(r *http.Request) zl.Logger {
	i := r.Context().Value(loggerKey{})
	if i != nil {
		l, ok := i.(zl.Logger)
		if ok {
			return l
		}
	}
	l := log.Default()
	l.Error().Msgf("zlogger missing from request context for %s (this is expected in unit tests)", r.URL)
	return l
}

// panicCatcher is http middleware that recovers from panics, logs them, and
// turns them into 500s.
func panicCatcher(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := logger(r)
		defer func() {
			err := recover()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				l.Error().Msgf("Handler panicked: %#v", err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// logHTTP is http middleware that dumps HTTP requests and responses via loghttp.
func logHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler := loghttp.Wrap(next, logger(r))
		handler.ServeHTTP(w, r)
	})
}
