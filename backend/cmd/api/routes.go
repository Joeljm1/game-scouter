package main

import (
	auth "game-scouter-api/handlers/authHandler"
	"game-scouter-api/internal/application"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type serverApplication struct {
	*application.Application
}

func (app *serverApplication) routes() http.Handler {
	mux := chi.NewRouter()

	// middlewares
	mux.Use(app.RecoverPanic)
	mux.Use(app.Metrics)
	mux.Get("/v1/health", app.healthCheck)

	// mount other handlers here
	authHander := auth.AuthApplication{Application: app.Application}
	mux.Mount("/auth", authHander.Routes())

	return mux
}
