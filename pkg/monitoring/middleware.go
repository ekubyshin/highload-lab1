package monitoring

import (
	"net/http"
	"time"
)

// PanicHandler writes the response after a panic. wroteHeader reports whether a
// final status was already committed; the handler must leave that status unchanged.
type PanicHandler func(w http.ResponseWriter, r *http.Request, recovered any, wroteHeader bool)

// statusRecorder remembers the first final HTTP status. Informational 1xx
// statuses are forwarded and do not become the recorded code. Later WriteHeader
// calls do not replace it.
type statusRecorder struct {
	http.ResponseWriter
	status    int
	committed bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if code >= 100 && code < 200 {
		s.ResponseWriter.WriteHeader(code)
		return
	}
	if s.committed {
		return
	}
	s.committed = true
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(p []byte) (int, error) {
	if !s.committed {
		s.committed = true
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(p)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

func (s *statusRecorder) statusCode() int {
	if !s.committed {
		return http.StatusOK
	}
	return s.status
}

func (m *Metrics) wrap(method, route string, next http.Handler, onPanic PanicHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		// observe is registered first so it runs after panic recovery and sees the final status.
		defer func() {
			m.observe(method, route, recorder.statusCode(), time.Since(start))
		}()
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			if onPanic == nil {
				panic(recovered)
			}
			onPanic(recorder, r, recovered, recorder.committed)
		}()
		next.ServeHTTP(recorder, r)
	})
}
