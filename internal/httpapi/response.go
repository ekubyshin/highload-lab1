package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
)

type errorBody struct {
	Error string `json:"error"`
}

type statusRecorder struct {
	http.ResponseWriter
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(status int) {
	if s.wroteHeader {
		return
	}
	s.wroteHeader = true
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(p []byte) (int, error) {
	if !s.wroteHeader {
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(p)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorBody{Error: message})
}
