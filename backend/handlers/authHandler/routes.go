package auth

import (
	"game-scouter-api/internal/validator"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *AuthApplication) Routes() http.Handler {
	r := chi.NewRouter()
	return r
}

func (app *AuthApplication) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req request
	err := app.ReadJSON(w, r, &req)
	if err != nil {
		app.BadReqResponse(w, r, err)
		return
	}
	v := validator.NewValidator()
	user := User{
		Name:      req.Name,
		Email:     req.Email,
		Activated: false,
	}
	ValidatePlaintext(v, req.Password)
	user.Validate(v)
	if !v.Valid() {
		app.ValidationErrResponse(w, r, v.Errors)
		return
	}
	psswd := password{}
	err = psswd.SetHash(req.Password)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
}
