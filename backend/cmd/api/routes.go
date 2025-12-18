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
	mux.Use(app.RateLimit)
	mux.Use(app.CheckCustomHeader)
	mux.Use(app.EnableCORS)
	mux.Use(app.PassLogger)
	mux.Use(app.Authenticate)
	mux.Get("/v1/health", app.healthCheck)

	// mount other handlers here
	mux.Mount("/auth", auth.Routes(app.Application))

	return mux
}
