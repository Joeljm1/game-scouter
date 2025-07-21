package main

import (
	"net/http"
)

func (app *application) healthCheck(w http.ResponseWriter, r *http.Request) {
	data := envelope{
		"Status": "Available",
		"ApiInfo": map[string]string{
			"Version":     version,
			"environment": app.cfg.env,
		},
	}
	err := app.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		app.serverErrResponse(w, r, err)
	}
}
