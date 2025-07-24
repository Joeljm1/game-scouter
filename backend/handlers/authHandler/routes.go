package auth

import (
	"crypto/sha256"
	"errors"
	"game-scouter-api/internal/application"
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/validator"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (app *AuthApplication) Routes() http.Handler {
	r := chi.NewRouter()
	r.HandleFunc("/register", app.RegisterUserHandler)
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
	user := data.User{
		Name:      req.Name,
		Email:     req.Email,
		Activated: false,
	}
	user.Validate(v)
	if !v.Valid() {
		app.ValidationErrResponse(w, r, v.Errors)
		return
	}
	psswd := data.Password{}
	err = psswd.SetHash(req.Password)
	if psswd.Hash == nil {
		panic("Hash not set")
	}
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	err = app.Models.UserModel.Insert(&user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUniqueViolation):
			v.AddError("email", "already exists")
			app.ValidationErrResponse(w, r, v.Errors)
		default:
			app.ServerErrResponse(w, r, err)
		}
		return
	}
	tok := data.GenerateToken(user.ID, time.Hour*48, data.ScopeActivation)
	err = app.Models.TokenModel.Insert(tok)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	tmplData := struct {
		userID          int64
		ActivationToken string
	}{
		userID:          user.ID,
		ActivationToken: tok.Plaintext,
	}
	app.Background(func() {
		i := 0
		for i < 3 {
			err := app.Mailer.Send(user.Email, tmplData)
			if err == nil {
				break
			}
		}

	})
	envelope := application.Envelope{
		"Successful": "Account created, now please verify mail",
	}
	err = app.WriteJSON(w, http.StatusCreated, envelope, nil) //TODO: check if status code that is sent is the valid one
	if err != nil {
		app.ServerErrResponse(w, r, err)
	}
}
func (app *AuthApplication) ActivateUser(w http.ResponseWriter, r *http.Request) {
	token := application.GetFromQuery(r, "token")
	v := validator.NewValidator()
	data.ValidateToken(v, token)
	if !v.Valid() {
		app.ValidationErrResponse(w, r, v.Errors)
		return
	}
	hashArr := sha256.Sum256([]byte(token))
	hash := hashArr[:]
	user, err := app.Models.UserModel.GetUserfromToken(hash, data.ScopeActivation)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRows):
			v.AddError("token", "not valid")
			app.ValidationErrResponse(w, r, v.Errors)
		default:
			app.ServerErrResponse(w, r, err)
		}
		return
	}
	user.Activated = true
	err = app.Models.UserModel.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrConflictFound):
			app.ConflictErrResponse(w, r)
		default:
			app.ServerErrResponse(w, r, err)
		}
		return
	}

	envelope := application.Envelope{
		"Successful": "Account activated",
	}
	err = app.WriteJSON(w, http.StatusOK, envelope, nil) //TODO: ceck if status code that is sent is the valid one
	if err != nil {
		app.ServerErrResponse(w, r, err)
	}
}
