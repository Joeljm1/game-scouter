package main

import (
	"game-scouter-api/internal/application"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type serverApplication struct {
	*application.Application
}

func (app *serverApplication) routes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(app.RecoverPanic)
	mux.Use(app.Metrics)
	mux.Get("/v1/health", app.healthCheck)

	return mux
}
