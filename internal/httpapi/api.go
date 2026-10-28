package httpapi

import (
	"errors"
	"log"
	"net/http"
	"runtime/debug"

	"highload/lab1/internal/item"
)

type repository interface {
	Create(name, value string) (item.Item, error)
	Get(id string) (item.Item, error)
	Update(id, name, value string) (item.Item, error)
}

type API struct {
	store        repository
	health       *Health
	maxBodyBytes int64
}

func New(store *item.Store, health *Health, maxBodyBytes int64) (*API, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}
	return newAPI(store, health, maxBodyBytes)
}

func newAPI(store repository, health *Health, maxBodyBytes int64) (*API, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}
	if health == nil {
		return nil, errors.New("health is required")
	}
	if maxBodyBytes <= 0 {
		return nil, errors.New("max body bytes must be positive")
	}
	return &API{store: store, health: health, maxBodyBytes: maxBodyBytes}, nil
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /items", a.create)
	mux.HandleFunc("GET /items/{id}", a.get)
	mux.HandleFunc("PUT /items/{id}", a.update)
	mux.Handle("GET /health", a.health)
	return withRecovery(mux)
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w}
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			log.Printf("http panic: %v\n%s", recovered, debug.Stack())
			if recorder.wroteHeader {
				return
			}
			writeError(recorder, http.StatusInternalServerError, "internal error")
		}()
		next.ServeHTTP(recorder, r)
	})
}
