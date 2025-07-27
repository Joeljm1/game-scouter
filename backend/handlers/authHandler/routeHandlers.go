package auth

//TODO: when frontend created redirect to home page

//TODO: restructuring of ActivateUserHandler as the order does not seem to be valid
// ie if token created but template execution error then token is simply created
// or not 🙂

// TODO: After making react appp may be change the
// response form email to react page instead of the tmpl one in ActivateUserHandler

import (
	"bytes"
	"errors"
	"game-scouter-api/internal/application"
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/validator"
	"net/http"
)

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
	user.Validate(v, req.Password)
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
	tok, err := app.Models.TokenModel.GenerateAndInsertToken(user.ID, app.Cfg.TokenLife.ActivateToken.LifeDuration, data.ScopeActivation)
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
	err = app.WriteJSON(w, http.StatusCreated, envelope, nil)
	if err != nil {
		app.ServerErrResponse(w, r, err)
	}
}

// TODO: After making react appp may be change the
// response form email to react page instead of the tmpl one
func (app *AuthApplication) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {
	token := application.GetFromQuery(r, "token")
	v := validator.NewValidator()
	data.ValidateToken(v, token)
	if !v.Valid() {
		app.ValidationErrResponse(w, r, v.Errors)
		return
	}
	user, err := app.Models.UserModel.GetUserfromToken(token, data.ScopeActivation)
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
	err = app.Models.TokenModel.DeleteAllToken(user.ID, data.ScopeActivation)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	// authToken, err := app.Models.TokenModel.GenerateAndInsertToken(user.ID, app.Cfg.TokenLife.AuthToken.LifeDuration, data.ScopeAuthentication)
	// if err != nil {
	// 	app.ServerErrResponse(w, r, err)
	// 	return
	// }
	// c := NewAuthTokenCookie(user.ID, authToken, app.Cfg.TokenLife.AuthToken.LifeDuration)
	buff := new(bytes.Buffer)
	tmplData := struct {
		Name string
	}{
		Name: user.Name,
	}
	err = welcomeTmpl.ExecuteTemplate(buff, "welcome", tmplData)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	//TODO: restructuring of this code as the order does not seem to be valid
	// ie if token created but template execution error then token is simply created
	// or not 🙂

	err = app.Login(w, user.ID, app.Cfg.TokenLife.ActivateToken.LifeDuration)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	// http.SetCookie(w, c)
	w.WriteHeader(http.StatusOK)
	w.Write(buff.Bytes())
}
func (app *AuthApplication) LoginHandler(w http.ResponseWriter, r *http.Request) {
	type request struct {
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
	data.ValidateEmail(v, req.Email)
	data.ValidatePlaintext(v, req.Password)
	if !v.Valid() {
		app.ValidationErrResponse(w, r, v.Errors)
		return
	}
	user, err := app.Models.UserModel.GetUserFromEmail(req.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRows):
			v.AddError("Error", "email not registered")
			app.ValidationErrResponse(w, r, v.Errors)
		default:
			app.ServerErrResponse(w, r, err)
		}
		return
	}
	match, err := data.MatchPassword(req.Password, user.Password.Hash)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	if !match {
		app.LoginErr(w, r)
		return
	}
	err = app.Login(w, user.ID, app.Cfg.TokenLife.AuthToken.LifeDuration)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	//TODO: when frontend created redirect to home page
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("This is homepage"))
}
