package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(app.RecoverPanic)
	mux.Use(app.Metrics)
	mux.Get("/v1/health", app.healthCheck)

	return mux
}
