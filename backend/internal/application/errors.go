package application

import (
	"log/slog"
	"net/http"
)

func (app *Application) LogErr(msg string, r *http.Request, err error) {
	app.Logger.Error(msg,
		slog.String("Error", err.Error()),
		slog.String("Method", r.Method),
		slog.String("Path", r.URL.String()))
}

func (app *Application) ErrResponse(w http.ResponseWriter, r *http.Request, status int, msg any) {
	data := Envelope{"Error": msg}
	err := app.WriteJSON(w, status, data, nil)
	if err != nil {
		app.LogErr("Server Error", r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *Application) ServerErrResponse(w http.ResponseWriter, r *http.Request, err error) {
	msg := "An internal server error has occured"
	app.LogErr(msg, r, err)
	app.ErrResponse(w, r, http.StatusInternalServerError, msg)
}

func (app *Application) BadReqResponse(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil { // just incase ig should never prolly happen
		app.ErrResponse(w, r, http.StatusBadRequest, err.Error())
	}
}

func (app *Application) RateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	msg := "Your rate limit has exceeded. Please try after some time"
	app.ErrResponse(w, r, http.StatusTooManyRequests, msg)
}
