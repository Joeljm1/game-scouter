package main

import (
	"game-scouter-api/internal/application"
	"net/http"
)

func (app *serverApplication) healthCheck(w http.ResponseWriter, r *http.Request) {
	data := application.Envelope{
		"Status": "Available",
		"ApiInfo": map[string]string{
			"Version":     version,
			"environment": app.Cfg.Env,
		},
	}
	err := app.WriteJSON(w, http.StatusOK, data, nil)
	if err != nil {
		app.ServerErrResponse(w, r, err)
	}
}
