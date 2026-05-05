package auth

//TODO: when frontend created redirect to home page

// TODO: After making react appp may be change the
// response form email to react page instead of the tmpl one in ActivateUserHandler

import (
	"errors"
	"fmt"
	"game-scouter-api/internal/application"
	oidc "game-scouter-api/internal/application/OIDC"
	"game-scouter-api/internal/application/OIDC/jwt"
	"game-scouter-api/internal/data"
	"game-scouter-api/internal/validator"
	"net/http"
	"net/url"

	"github.com/jackc/pgx/v5"
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
	user := data.ValidateUser(v, req.Name, req.Email, req.Password)
	if !v.Valid() {
		app.ValidationErrResponse(w, r, v.Errors)
		return
	}
	psswd := data.Password{}
	err = psswd.SetHash(req.Password)
	if err != nil || psswd.Hash == nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	user.Password = psswd
	err = app.Models.UserModel.InsertUser(r.Context(), user)
	if err != nil {
		switch {
		// dont think [data.ErrUniqueViolation] will ever come may remove it later after checking
		case errors.Is(err, data.ErrUniqueViolation), errors.Is(err, data.ErrUserExists):
			v.AddError("email", "already exists")
			app.ValidationErrResponse(w, r, v.Errors)
		default:
			app.ServerErrResponse(w, r, err)
		}
		return
	}
	tok, err := app.Models.TokenModel.GenerateAndInsertToken(r.Context(), user.ID, app.Cfg.TokenLife.ActivateToken.LifeDuration, data.ScopeActivation)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	tmplData := struct {
		UserID          int64
		ActivationToken string
	}{
		UserID:          user.ID,
		ActivationToken: tok.Plaintext,
	}
	app.Background(func() {
		for i := range 3 {
			err := app.Mailer.Send(user.Email, tmplData)
			app.Logger.Info("Mailer.Send", "err", err)
			if err == nil {
				app.Logger.Info("Successfully sent email to", "email", user.Email)
				break
			}
			if i == 2 {
				app.Logger.Info("could not send email to", "email", user.Email)
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
	// not put in cache so get directly also getting session data i doubt is bad cause it will always be empty
	user, _, scope, err := app.Models.UserModel.GetUserfromTokenWithSess(r.Context(), token)
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
	if scope != data.ScopeActivation {
		app.LogErr("In Activation handler scope gotten from db for token was not Activation but %v", r, nil)
		app.BadReqResponse(w, r, errors.New("invalid token"))
		return
	}
	user.Activated = true
	err = app.Models.UserModel.Update(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrConflictFound):
			app.ConflictErrResponse(w, r)
		default:
			app.ServerErrResponse(w, r, err)
		}
		return
	}
	err = app.Models.TokenModel.DeleteAllToken(r.Context(), user.ID, data.ScopeActivation)
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
	//TODO: welcome mail???
	buff, err := WelcomeHTML(user.Name)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	//TODO: restructuring of this code as the order does not seem to be valid
	// ie if token created but template execution error then token is simply created
	// or not 🙂

	err = app.Login(r.Context(), w, user.ID, data.ScopeAuthentication)
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
	user, err := app.Models.UserModel.GetUserFromEmail(r.Context(), req.Email)
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
	//TODO: check for activation
	match, err := data.MatchPassword(req.Password, user.Password.Hash)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	if !match {
		app.LoginErr(w, r)
		return
	}
	err = app.Login(r.Context(), w, user.ID, data.ScopeAuthentication)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	//TODO: when frontend created redirect to home page
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("This is homepage"))
}

// redirect in url is hardcoded
func (app *AuthApplication) getGoogleOidcUrlHandler(w http.ResponseWriter, r *http.Request) {
	tok := app.GetTok(r)
	state := app.CryptoRandomStr(16)
	nonce := app.CryptoRandomStr(16)
	err := app.SetOIDCState(r, tok, state)
	if err != nil {
		// switch {
		// case errors.Is(err, pgx.ErrNoRows):
		// 	app.BadTokenResponse(w, r)
		// default:
		// 	app.ServerErrResponse(w, r, err)
		// }
		app.ServerErrResponse(w, r, err)
		return
	}
	err = app.SetOIDCNonce(r, tok, nonce)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	halfUrl := "%v?client_id=%v&response_type=code&state=%v&scope=openid%%20profile%%20email&redirect_uri=%v&nonce=%v"
	authEndpoint := app.Cfg.OIDC.Google.DocumentDiscovery.AuthorizationEndpoint
	url := fmt.Sprintf(halfUrl, authEndpoint, app.Cfg.OIDC.Google.ClientID, state, app.Cfg.OIDC.Google.OIDCRedirectURL, nonce)
	err = app.WriteJSON(w, http.StatusOK, application.Envelope{"url": url}, nil)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
}

type TokenResp struct {
	Token  string `json:"id_token"`
	Expire int    `json:"expires_in"`
}

func (app *AuthApplication) googleOIDCRedirectHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		app.BadReqResponse(w, r, errors.New("missing code/state as URL param"))
		return
	}
	ok, err := app.VerifyOIDCState(r, state)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			app.NotAuthenticatedResponse(w, r)
		default:
			app.ServerErrResponse(w, r, err)
		}
		return
	}

	if !ok {
		app.BadReqResponse(w, r, errors.New("invalid OIDC State"))
		return
	}

	// send data to google enpoint and get jwt
	tokenUrl := app.Cfg.OIDC.Google.DocumentDiscovery.TokenEndpoint
	query := url.Values{}
	query.Set("code", code)
	query.Set("client_id", app.Cfg.OIDC.Google.ClientID)
	query.Set("client_secret", app.Cfg.OIDC.Google.ClientSecret)
	query.Set("redirect_uri", app.Cfg.OIDC.Google.OIDCRedirectURL)
	query.Set("grant_type", "authorization_code")

	resp, err := app.HttpClient.PostForm(tokenUrl, query)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	defer resp.Body.Close()
	oResp, err := oidc.NewResp(resp.Body)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	JWT, err := jwt.New(oResp.IDToken)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	valid, payload, err := app.Cfg.OIDC.Google.Verify(JWT)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	if !valid {
		app.BadReqResponse(w, r, errors.New("JWT is not valid"))
		return
	}
	valid, err = app.VerifyOIDCNonceAndEmail(r, payload.Nonce, payload.Email, payload.Email_verified)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailInvalid):
			app.BadReqResponse(w, r, errors.New("invalid/unverified email"))
		default:
			app.ServerErrResponse(w, r, err)
		}
		return
	}
	if !valid {
		app.BadReqResponse(w, r, errors.New("invalid request"))
		return
	}
	user, err := app.Models.UserModel.GetUserFromEmail(r.Context(), *payload.Email)
	if err != nil { // no user with that
		switch {
		case errors.Is(err, data.ErrNoRows):
			var name string
			if payload.Name == nil {
				name = ""
			} else {
				name = *payload.Name
			}
			user := data.User{
				Name:      name,
				Email:     *payload.Email,
				Activated: false, // just to make it explicit so i dont forget later
			}
			// [data.UserModel.InsertUser] not used here because we already know email is not there in table
			// so no need to check again
			err = app.Models.UserModel.Insert(r.Context(), &user)
			if err != nil {
				app.ServerErrResponse(w, r, err)
			}
			buff, err := WelcomeHTML(name)
			//TODO: welcome mail???
			if err != nil {
				app.ServerErrResponse(w, r, err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write(buff.Bytes())
		default:
			app.ServerErrResponse(w, r, err)
		}
		return

	}
	err = app.Login(r.Context(), w, user.ID, data.ScopeOIDC)
	if err != nil {
		app.ServerErrResponse(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("This is homepage"))
}
