/*
Package api provides route definitions and handling
*/
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func NewRouter() http.Handler {
	r := chi.NewRouter()
	// middleware
	r.Get("/healthz", handleHealth)
	return r
}
