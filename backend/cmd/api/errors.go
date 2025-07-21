package main

import (
	"log/slog"
	"net/http"
)

func (app *application) logErr(msg string, r *http.Request, err error) {
	app.logger.Error(msg,
		slog.String("Error", err.Error()),
		slog.String("Method", r.Method),
		slog.String("Path", r.URL.String()))
}

func (app *application) errResponse(w http.ResponseWriter, r *http.Request, status int, msg any) {
	data := envelope{"Error": msg}
	err := app.writeJSON(w, status, data, nil)
	if err != nil {
		app.logErr("Server Error", r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *application) serverErrResponse(w http.ResponseWriter, r *http.Request, err error) {
	msg := "An internal server error has occured"
	app.logErr(msg, r, err)
	app.errResponse(w, r, http.StatusInternalServerError, msg)
}

func (app *application) badReqResponse(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil { // just incase ig should never prolly happen
		app.errResponse(w, r, http.StatusBadRequest, err.Error())
	}
}

func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	msg := "Your rate limit has exceeded. Please try after some time"
	app.errResponse(w, r, http.StatusTooManyRequests, msg)
}
