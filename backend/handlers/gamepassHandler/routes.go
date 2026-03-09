package gamepass

import (
	"game-scouter-api/internal/application"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type GamepassApplication struct {
	*application.Application
	AllGames *AllGames
}

func (app *GamepassApplication) routes() http.Handler {
	r := chi.NewRouter()
	// r.Get()
	return r
}
