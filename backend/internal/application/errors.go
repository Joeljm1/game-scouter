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

// would this cause a looop???
func (app *Application) ErrResponse(w http.ResponseWriter, r *http.Request, status int, msg any) {
	data := Envelope{"Error": msg}
	err := app.WriteJSON(w, status, data, nil)
	if err != nil {
		app.ServerErrResponse(w, r, err)
	}
}

func (app *Application) ServerErrResponse(w http.ResponseWriter, r *http.Request, err error) {
	msg := "An internal server error has occured"
	app.LogErr(msg, r, err)
	app.ErrResponse(w, r, http.StatusInternalServerError, msg)
}

// Make sure error is not nil
func (app *Application) BadReqResponse(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil { // just incase ig should never prolly happen
		app.ErrResponse(w, r, http.StatusBadRequest, err.Error())
	}
}

func (app *Application) RateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	msg := "Your rate limit has exceeded. Please try after some time"
	app.ErrResponse(w, r, http.StatusTooManyRequests, msg)
}

func (app *Application) ValidationErrResponse(w http.ResponseWriter, r *http.Request, errMap map[string]string) {
	data := Envelope{"Error": errMap}
	// err := app.WriteJSON(w, http.StatusBadRequest, data, nil)
	// if err != nil {
	// 	app.ServerErrResponse(w, r, err)
	// }
	app.ErrResponse(w, r, http.StatusBadRequest, data)
}

func (app *Application) ConflictErrResponse(w http.ResponseWriter, r *http.Request) {
	msg := "unable to update the record due to an edit conflict, please try again"
	app.ErrResponse(w, r, http.StatusConflict, msg)
}

func (app *Application) LoginErr(w http.ResponseWriter, r *http.Request) {
	msg := "invalid credentials"
	app.ErrResponse(w, r, http.StatusUnauthorized, msg)
}

func (app *Application) NotAuthenticatedResponse(w http.ResponseWriter, r *http.Request) {
	msg := "user not authenticated"
	app.ErrResponse(w, r, http.StatusForbidden, msg)
}

func (app *Application) NotActivatedResponse(w http.ResponseWriter, r *http.Request) {
	msg := "user not activated"
	app.ErrResponse(w, r, http.StatusForbidden, msg)
}

func (app *Application) BadTokenResponse(w http.ResponseWriter, r *http.Request) {
	msg := "bad token recived"
	app.ErrResponse(w, r, http.StatusBadRequest, msg)
}
